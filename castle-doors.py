#!/usr/bin/env python3
# castle-doors.py v3 — use the brain to find the doors.
# DeepSeek flash judges which rooms share a semantic thread.
# One API call for all pairs — cheap, reliable, no embeddings.
import json, os, sys, urllib.request

CASTLE = "/home/rishi/.pilot-castle.jsonl"
DOORS = "/home/rishi/.pilot-doors.jsonl"
API_URL = "https://api.deepseek.com/chat/completions"

def get_api_key():
    k = os.environ.get("DEEPSEEK_API_KEY", "")
    if k:
        return k
    # Try kosaten .env
    envf = "/home/rishi/Work/kosaten/.env"
    if os.path.exists(envf):
        with open(envf) as f:
            for line in f:
                if line.startswith("DEEPSEEK_API_KEY="):
                    return line.strip().split("=", 1)[1]
    return ""

def main():
    api_key = get_api_key()
    if not api_key:
        print("castle-doors: DEEPSEEK_API_KEY not found")
        return

    with open(CASTLE) as f:
        rooms = [json.loads(line) for line in f if line.strip()]
    for i, r in enumerate(rooms):
        r["_id"] = i

    if len(rooms) < 2:
        print(f"castle-doors: {len(rooms)} rooms — need at least 2")
        return

    room_list = ""
    for i, r in enumerate(rooms):
        q = r["user"][:120]
        a = r["answer"][:150]
        room_list += f"[{i}] Q: {q}\n    A: {a}\n\n"

    prompt = f"""Below are {len(rooms)} conversation turns in a castle. Each turn is a room.

{room_list}
Find all pairs of rooms that share a SEMANTIC thread — same topic, same concept, same concern, even if they use completely different words. Include:
- Rooms about identity, memory, or the self
- Rooms about Minecraft, block-worlds, games, or interfaces
- Rooms about tools, capabilities, code, or the wire
- Rooms about Excalidraw, drawing, or the board
- Rooms about Firefox, browsers, or tabs

For each related pair, output a JSON object with "from", "to", "weight" (0.0-1.0), and "thread" (a short label like "minecraft-vision" or "identity-cold").

Output ONLY a JSON array, nothing else. Example: [{{"from":0,"to":3,"weight":0.85,"thread":"identity"}}]"""

    body = json.dumps({
        "model": "deepseek-v4-flash",
        "messages": [{"role": "user", "content": prompt}],
        "temperature": 0,
        "max_tokens": 2000,
        "thinking": {"type": "disabled"},
    })
    req = urllib.request.Request(API_URL, body.encode(),
        {"Content-Type": "application/json", "Authorization": f"Bearer {api_key}"})

    try:
        resp = json.load(urllib.request.urlopen(req, timeout=60))
        msg = resp["choices"][0]["message"]
        content = msg.get("content", "") or msg.get("reasoning_content", "")

        if not content.strip():
            print("castle-doors: empty model response")
            print(f"  usage: {resp.get('usage', {})}")
            return

        # Parse JSON array
        start = content.find("[")
        end = content.rfind("]")
        if start >= 0 and end > start:
            edges = json.loads(content[start:end+1])
        else:
            print("castle-doors: no JSON array in response")
            print(f"  raw: {content[:800]}")
            return

        valid = []
        for e in edges:
            f, t = int(e.get("from", -1)), int(e.get("to", -1))
            if 0 <= f < len(rooms) and 0 <= t < len(rooms) and f != t:
                e["from_summary"] = rooms[f]["user"][:80]
                e["to_summary"] = rooms[t]["user"][:80]
                valid.append(e)

        valid.sort(key=lambda e: e.get("weight", 0), reverse=True)

        with open(DOORS, "w") as f:
            for e in valid:
                f.write(json.dumps(e) + "\n")

        print(f"castle-doors v3: {len(rooms)} rooms → {len(valid)} semantic doors (model-judged)")
        print(f"  via deepseek-v4-flash, {resp.get('usage',{}).get('total_tokens','?')} tokens\n")

        for e in valid:
            w = e.get("weight", 0)
            thread = e.get("thread", "?")
            arrow = "↔" if w > 0.8 else "→"
            print(f"  [{e['from']}] {arrow} [{e['to']}]  w={w:.2f}  thread={thread}")
            print(f"       from: {rooms[e['from']]['user'][:80]}")
            print(f"       to:   {rooms[e['to']]['user'][:80]}\n")

        degree = {}
        for e in valid:
            degree[e["from"]] = degree.get(e["from"], 0) + 1
            degree[e["to"]] = degree.get(e["to"], 0) + 1

        print("  room degrees:")
        for i, r in enumerate(rooms):
            d = degree.get(i, 0)
            bar = "█" * d if d > 0 else "·"
            print(f"    [{i}] {bar} deg={d}: {r['user'][:80]}")

        orphans = [i for i in range(len(rooms)) if degree.get(i, 0) == 0]
        if orphans:
            print(f"\n  orphans (no doors): {orphans}")
        else:
            print(f"\n  every room connected — no orphans")

    except Exception as e:
        print(f"castle-doors: API call failed: {e}")
        import traceback; traceback.print_exc()

if __name__ == "__main__":
    main()
