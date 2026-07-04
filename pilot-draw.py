#!/usr/bin/env python3
# pilot-draw.py — pilot's hand for the shared Excalidraw board.
#
# Give it a JSON spec: {"title": "...", "boxes": [{"label","body"}...], "note": "...", "x": 5200, "y": 60}
# It injects a clean titled diagram onto the board through the wire (:4445 broker ->
# the excalidraw tab's own updateScene). Placed in free space to the right.
#
#   python3 pilot-draw.py '{"title":"How pilot works","boxes":[{"label":"BRAIN","body":"DeepSeek flash/pro"}],"x":5200,"y":60}'
import sys, json, urllib.request, random

BROKER = "http://127.0.0.1:4445/command"


def cmd(m, p, tmo=30):
    req = urllib.request.Request(BROKER, json.dumps({"method": m, "params": p}).encode(),
                                 {"Content-Type": "application/json"})
    return json.load(urllib.request.urlopen(req, timeout=tmo))


def ctxs():
    out = {}
    def walk(cs):
        for c in cs:
            out[c["context"]] = c.get("url", "")
            walk(c.get("children", []))
    walk(cmd("browsingContext.getTree", {})["result"]["contexts"])
    return out


def main():
    exc = [c for c, u in ctxs().items() if "excalidraw" in u]
    if not exc:
        print("pilot-draw: no excalidraw tab open on the peer")
        sys.exit(1)
    CTX = exc[0]
    spec = json.loads(sys.argv[1] if len(sys.argv) > 1 else sys.stdin.read())
    title = spec.get("title", "untitled")
    boxes = spec.get("boxes", [])
    note = spec.get("note", "")

    PAL = [("#e7f5ff", "#1971c2"), ("#f3f0ff", "#6741d9"), ("#ebfbee", "#2f9e44"),
           ("#fff4e6", "#e8590c"), ("#fff0f6", "#c2255c"), ("#fff9db", "#e67700")]
    base = dict(angle=0, fillStyle="solid", strokeWidth=1, strokeStyle="solid", roughness=1,
                opacity=100, isDeleted=False, boundElements=[], updated=1, link=None,
                locked=False, groupIds=[], frameId=None)
    seed = [random.randint(10**7, 10**8)]  # unique per run — repeated draws must not collide ids (else updateScene overwrites the previous diagram)

    def sn():
        seed[0] += 7
        return seed[0]

    def rect(x, y, w, h, bg, st):
        e = dict(base)
        e.update(type="rectangle", id=f"pd_r_{sn()}", x=x, y=y, width=w, height=h,
                 strokeColor=st, backgroundColor=bg, seed=sn(), version=1, versionNonce=sn(),
                 roundness={"type": 3})
        return e

    def text(x, y, w, h, s, st, fs=13, fam=2):
        e = dict(base)
        e.update(type="text", id=f"pd_t_{sn()}", x=x, y=y, width=w, height=h, strokeColor=st,
                 backgroundColor="transparent", seed=sn(), version=1, versionNonce=sn(), text=s,
                 fontSize=fs, fontFamily=fam, textAlign="left", verticalAlign="top",
                 baseline=int(fs * 0.9), containerId=None, originalText=s, lineHeight=1.25)
        return e

    X0 = spec.get("x", 5200)
    Y0 = spec.get("y", 60)
    W = 540
    els = [text(X0, Y0, W + 200, 40, title, "#1e1e1e", fs=26)]
    if note:
        els.append(text(X0, Y0 + 40, W + 300, 22, note, "#666", fs=13, fam=3))
    y = Y0 + 100
    for i, b in enumerate(boxes):
        body = b.get("body", "")
        lines = body.count("\n") + 1 if body else 0
        h = 44 + lines * 18 + 8
        bg, st = PAL[i % len(PAL)]
        els.append(rect(X0, y, W, h, bg, st))
        els.append(text(X0 + 12, y + 9, W - 20, 26, b.get("label", ""), st, fs=15, fam=2))
        if body:
            els.append(text(X0 + 14, y + 38, W - 24, lines * 18, body, "#1e1e1e", fs=12, fam=3))
        y += h + 26

    payload = json.dumps(els)
    expr = ("(()=>{const root=document.querySelector('.excalidraw');"
            "const key=Object.keys(root).find(k=>k.startsWith('__reactFiber$'));"
            "let stack=[root[key]],seen=new Set(),api=null,hops=0;"
            "while(stack.length&&hops<40000){const n=stack.pop();hops++;if(!n||seen.has(n))continue;seen.add(n);"
            "for(const o of [n.memoizedState,n.memoizedProps,n.stateNode]){if(o&&typeof o==='object'){"
            "if(typeof o.getSceneElements==='function'&&typeof o.updateScene==='function'){api=o;}}}"
            "if(api)break;if(n.child)stack.push(n.child);if(n.sibling)stack.push(n.sibling);if(n.return&&!seen.has(n.return))stack.push(n.return);}"
            "if(!api)return 'no-api';const cur=api.getSceneElements();const add=%s;"
            "api.updateScene({elements:cur.concat(add)});api.scrollToContent(add,{fitToContent:true});"
            "return 'drew '+add.length+' elements';})()") % payload
    r = cmd("script.evaluate", {"target": {"context": CTX}, "awaitPromise": True, "expression": expr})
    print("pilot-draw:", r.get("result", {}).get("result", {}).get("value", r))


if __name__ == "__main__":
    main()
