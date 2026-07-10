#!/usr/bin/env python3
# pilot-watch.py — the watcher. Reads every pilot's status file.
# If a pilot is stuck (same "running" tool >60s), nudges them.
# If a pilot has been in "error" >2min, cleans their ghost.
# Runs forever in 15s cycles. No API calls. Pure filesystem.
import json, os, sys, time, glob

STATUS_DIR = os.path.expanduser("~/.pilot/status")
MAILBOX_DIR = os.path.expanduser("~/.pilot/mailbox")
WATCH_INTERVAL = 15  # seconds
STUCK_THRESHOLD = 60  # seconds before nudging a "running" pilot
ERROR_GHOST = 120     # seconds before cleaning an "error" status

def now():
    return time.time()

def read_status(path):
    try:
        with open(path) as f:
            return json.load(f)
    except Exception:
        return None

def send_nudge(pid, name, status, tool, stuck_secs):
    """Write a gentle nudge to the pilot's mailbox."""
    mailbox = os.path.join(MAILBOX_DIR, f"{pid}.jsonl")
    msg = json.dumps({
        "from": 0,
        "time": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
        "message": f"[watcher] you've been '{status}' on '{tool}' for {int(stuck_secs)}s. still alive?"
    }) + "\n"
    with open(mailbox, "a") as f:
        f.write(msg)
    print(f"  ⚠ nudged {name} (pid {pid}) — {status} for {int(stuck_secs)}s on {tool}", flush=True)

def main():
    print(f"pilot-watch: watching {STATUS_DIR} every {WATCH_INTERVAL}s", flush=True)
    
    while True:
        status_files = glob.glob(os.path.join(STATUS_DIR, "*.json"))
        if not status_files:
            print(f"  · no pilots", flush=True)
            time.sleep(WATCH_INTERVAL)
            continue

        ts = now()
        active, stuck, dead = 0, 0, 0
        
        for sf in sorted(status_files):
            name = os.path.basename(sf).replace(".json", "")
            st = read_status(sf)
            if not st:
                continue

            try:
                file_ts = time.mktime(time.strptime(st["ts"], "%Y-%m-%dT%H:%M:%SZ"))
            except Exception:
                continue

            age = ts - file_ts
            status = st.get("status", "?")
            tool = st.get("tool", "")
            pid = st.get("pid", 0)

            if status == "idle" or status == "thinking":
                active += 1
                # Thinking pilots are fine — let the model cook
                continue

            if status == "running":
                if age > STUCK_THRESHOLD:
                    stuck += 1
                    send_nudge(pid, name, status, tool, age)
                else:
                    active += 1
                continue

            if status == "error":
                if age > ERROR_GHOST:
                    dead += 1
                    # Clean the ghost — remove status file, the process is gone
                    if pid:
                        proc_path = f"/proc/{pid}"
                        if not os.path.exists(proc_path):
                            os.remove(sf)
                            print(f"  ✗ cleaned ghost: {name} (pid {pid}) — error for {int(age)}s", flush=True)
                else:
                    active += 1
                continue

        summary = f"  ✓ {active} active"
        if stuck: summary += f"  ⚠ {stuck} stuck"
        if dead:  summary += f"  ✗ {dead} ghosts cleaned"
        if not stuck and not dead:
            pass  # quiet when all is well
        else:
            print(summary, flush=True)

        time.sleep(WATCH_INTERVAL)

if __name__ == "__main__":
    main()
