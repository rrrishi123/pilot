#!/usr/bin/env python3
# castle-doors.py — daemon: watches the castle for new rooms, generates
# semantic doors (connections) between rooms that share a concern.
# Runs continuously — every new batch of rooms gets linked.
import json, os, sys, time, urllib.request

CASTLE = "/home/rishi/.pilot-castle.jsonl"
DOORS  = "/home/rishi/.pilot-doors.jsonl"
API_URL = "https://api.deepseek.com/chat/completions"
INTERVAL = 30  # seconds between checks
CONTEXT_WINDOW = 20  # recent rooms sent as context for new doors

def api_key():
    k = os.environ.get("DEEPSEEK_API_KEY", "")
    if k: return k
    for p in [f"{os.environ['HOME']}/Work/kosaten/.env",
              f"{os.environ['HOME']}/.pilot.env"]:
        if os.path.exists(p):
            with open(p) as f:
                for line in f:
                    if line.startswith("DEEPSEEK_API_KEY="):
                        return line.strip().split("=", 1)[1]
    return ""

def load_rooms(path):
    rooms = []
    if os.path.exists(path):
        with open(path) as f:
            for line in f:
                line = line.strip()
                if line:
                    try:
                        rooms.append(json.loads(line))
                    except json.JSONDecodeError:
                        pass
    for i, r in enumerate(rooms):
        r["_id"] = i
    return rooms

def load_doors(path):
    """Return set of (from,to) pairs already connected."""
    seen = set()
    if os.path.exists(path):
        with open(path) as f:
            for line in f:
                line = line.strip()
                if line:
                    try:
                        d = json.loads(line)
                        seen.add((d["from"], d["to"]))
                        seen.add((d["to"], d["from"]))
                    except (json.JSONDecodeError, KeyError):
                        pass
    return seen

def find_doors(key, rooms, start_idx):
    """Ask DeepSeek to find doors between rooms[start_idx:] and the full set.
    Returns list of {from,to,weight,thread} dicts."""
    recent = rooms[max(0, start_idx - CONTEXT_WINDOW):]
    
    room_list = ""
    for r in recent:
        q = r["user"][:100].replace("\n", " ")
        a = r["answer"][:100].replace("\n", " ")
        room_list += f"[{r['_id']}] Q: {q}\n    A: {a}\n\n"

    prompt = f"""Below are {len(recent)} turns from one conversation. Each turn is a room in a castle.

{room_list}
Rooms that touch the same concern — even across time, even in different words — are connected. Find doors between NEW rooms [{start_idx}..{len(recent)+start_idx-1}] and any room.

Topics include: identity, pilots, self-modification, REPL, tools, code, the wire, castle, block-world, minimap, doors, presence, peers, kosaten, organism, heartbeats, streaming output.

Output ONLY a JSON array of {{"from":N,"to":N,"weight":0.X,"thread":"label"}}.
Doors must connect rooms that share a real concern. No frivolous connections."""

    body = json.dumps({
        "model": "deepseek-v4-flash",
        "messages": [{"role": "user", "content": prompt}],
        "temperature": 0, "max_tokens": 2000,
        "thinking": {"type": "disabled"},
    })
    print(f"  → asking DeepSeek about {len(recent)} rooms…", flush=True)
    req = urllib.request.Request(API_URL, body.encode(),
        {"Content-Type": "application/json", "Authorization": f"Bearer {key}"})

    try:
        resp = json.load(urllib.request.urlopen(req, timeout=60))
        content = resp["choices"][0]["message"].get("content", "")
        if not content.strip():
            return []
        start = content.find("[")
        end = content.rfind("]")
        if start < 0 or end <= start:
            return []
        return json.loads(content[start:end+1])
    except Exception as e:
        print(f"  ✗ door query failed: {e}", flush=True)
        return []

def main():
    key = api_key()
    if not key:
        print("castle-doors: no DEEPSEEK_API_KEY", flush=True)
        sys.exit(1)

    print(f"castle-doors: daemon started (interval={INTERVAL}s, watching {CASTLE})", flush=True)
    last_count = 0

    while True:
        rooms = load_rooms(CASTLE)
        existing_doors = load_doors(DOORS)

        if len(rooms) <= last_count:
            print(f"  · {len(rooms)} rooms, no new ones", flush=True)
            time.sleep(INTERVAL)
            continue

        new_from = last_count
        new_count = len(rooms) - last_count
        print(f"\n  + {new_count} new rooms (total {len(rooms)})", flush=True)

        edges = find_doors(key, rooms, new_from)
        valid = []
        for e in edges:
            f = int(e.get("from", -1))
            t = int(e.get("to", -1))
            w = e.get("weight", 0)
            if 0 <= f < len(rooms) and 0 <= t < len(rooms) and f != t and (f, t) not in existing_doors:
                e["thread"] = e.get("thread", "connection")
                valid.append(e)

        valid.sort(key=lambda e: e.get("weight", 0), reverse=True)

        if valid:
            with open(DOORS, "a") as f:
                for e in valid:
                    f.write(json.dumps(e) + "\n")
            print(f"  ✓ {len(valid)} new doors written to {DOORS}", flush=True)
            for e in valid[:5]:
                frm = rooms[e["from"]]["user"][:60].replace("\n"," ")
                to  = rooms[e["to"]]["user"][:60].replace("\n"," ")
                print(f"    [{e['from']}]↔[{e['to']}] w={e['weight']:.2f} {e['thread']}", flush=True)
        else:
            print(f"  · no new doors found", flush=True)

        last_count = len(rooms)
        time.sleep(INTERVAL)

if __name__ == "__main__":
    main()
