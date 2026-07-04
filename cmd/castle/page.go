package main

// The world page. One canvas, no assets, no build step. The palette and the
// region names are pilot's (chosen in the design conversation); the parchment
// and amber are the comic's DNA so the whole box reads as one civilisation.
// Gentle by contract: ~10fps, and it sleeps entirely when the tab is hidden.
const page = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>pilot castle · the world</title>
<style>
  html,body{margin:0;height:100%;background:#0e0b03;overflow:hidden}
  canvas{display:block;width:100vw;height:100vh}
  #say{position:fixed;left:50%;bottom:14px;transform:translateX(-50%);width:44vw;
    background:#1d1509;border:1px solid #6a4818;border-radius:6px;color:#d4b464;
    font:13px Georgia,serif;padding:8px 12px;outline:none}
  #say::placeholder{color:#786030}
</style>
</head>
<body>
<canvas id="w"></canvas>
<input id="say" placeholder="speak into the world — pilot answers here, kosaten through pilot, claude when it wakes" autocomplete="off">
<script>
"use strict";
// ---- palette (pilot's table) ----
var C = {
  bg:      "#0e0b03",
  path:    "#2a2210",
  gate:    "#3a3528", hall: "#2a2210", arm: "#1a1814",
  lib:     "#1e1a12", forge:"#221510", spire:"#14100a", court:"#1c180c",
  text:    "#d4b464", mid:  "#a88040", dim: "#786030",
  accent:  "#c07828", hi:   "#e0a040",
  pilotAv: "#e0a040",           // amber gold — the one who lives inside
  claudeAv:"#b8ccd8",           // pale blue-white — the one on the wire
  wire:    "#1a1c20",           // pale-blue-under-charcoal — the wire house ground
  good:    "#7a9a50", bad: "#a04828"
};
var cv = document.getElementById("w"), cx = cv.getContext("2d");
var W=0, H=0, DPR = Math.min(window.devicePixelRatio||1, 2);
function resize(){ W=innerWidth; H=innerHeight; cv.width=W*DPR; cv.height=H*DPR; cx.setTransform(DPR,0,0,DPR,0,0); }
resize(); addEventListener("resize", resize);

// ---- world state ----
var rooms=[], doors=[], presence={}, comic="", health=null, houses=0;
var drops={};                    // room i -> birth time (construction animation)
var bubbles=[];                  // {who,text,t0}
var av = {};                      // avatars slide, never teleport — created on demand
var AVCOL = { claude:C.claudeAv };
var NEXTCOL = ["#e0a040","#b8ccd8","#d8c8a0","#80c0a0","#c080d0","#e0a060","#80b0d0","#d0a080","#a0c080","#c0a0e0"];
var COLI = 0;
function avatarFor(who){         // anyone who speaks or has presence gets a body
  if(!av[who]){ av[who]={x:W*0.5,y:H*0.86,tx:W*0.5,ty:H*0.86}; }
  if(!AVCOL[who]){ AVCOL[who]=NEXTCOL[COLI%NEXTCOL.length]; COLI++; }
  return av[who];
}
// seed the standard two
avatarFor("pilot"); AVCOL.pilot=C.pilotAv;
avatarFor("claude"); AVCOL.claude=C.claudeAv;

// ---- view transform: fit the village into the middle of the screen ----
var view = {s:18, ox:0, oy:0};
function refit(){
  if(!rooms.length) return;
  var x0=1e9,y0=1e9,x1=-1e9,y1=-1e9;
  rooms.forEach(function(r){ x0=Math.min(x0,r.x); y0=Math.min(y0,r.y); x1=Math.max(x1,r.x); y1=Math.max(y1,r.y); });
  var span = Math.max(x1-x0+4, y1-y0+4);
  view.s = Math.max(9, Math.min(24, Math.min(W*0.60, H*0.72)/span));
  view.ox = W*0.46 - (x0+x1)/2*view.s;
  view.oy = H*0.52 - (y0+y1)/2*view.s;
}
function sx(x){ return view.ox + x*view.s; }
function sy(y){ return view.oy + y*view.s; }

function roomPos(i){ i=Math.max(0,Math.min(rooms.length-1,i|0)); var r=rooms[i]; return r?{x:sx(r.x),y:sy(r.y)}:{x:W/2,y:H/2}; }
function moveAvatar(who,i){ var a=avatarFor(who); var p=roomPos(i); a.tx=p.x; a.ty=p.y; if(!a.x&&!a.y){a.x=p.x;a.y=p.y;} }

// ---- data in ----
fetch("/world.json").then(function(r){return r.json();}).then(function(w){
  rooms=w.rooms||[]; doors=w.doors||[]; presence=w.presence||{}; comic=w.comic||""; health=w.health||null; houses=w.houses||0;
  refit();
  Object.keys(presence).forEach(function(k){ moveAvatar(k, presence[k]); });
});
var es = new EventSource("/events");
es.onmessage = function(m){
  var e; try{ e=JSON.parse(m.data); }catch(_){ return; }
  if(e.type==="room"){
    rooms[e.i]={u:e.u,a:e.a,x:e.x,y:e.y}; drops[e.i]=now(); refit();
    if(e.u) bubbles.push({who:"claude", text:e.u, t0:now()});
    if(e.a) bubbles.push({who:"pilot",  text:e.a, t0:now()+900});
    moveAvatar("pilot", e.i);
    if(now()-lastSpeech < 60000) crossings.push({i:e.i, t0:now()}); // both strands awake — a crossing
  }
  if(e.type==="presence"){ presence=e.who||{}; Object.keys(presence).forEach(function(k){ moveAvatar(k, presence[k]); }); }
  if(e.type==="comic"){ comic=e.text||""; }
  if(e.type==="health"){ health=e.h; }
  if(e.type==="houses"){ houses=e.n||0; }
  if(e.type==="speech"){ // a mind talking in real time — tail or commons
    var who=e.who||"claude";
    if(who==="claude") lastSpeech=now();
    bubbles.push({who:who, text:e.text, t0:now()});
    var a=avatarFor(who);
    if(who==="claude"){ a.tx=homeX(); a.ty=homeY(); } // claude walks home to speak
  }
};

// ---- the wire house: claude's street. One pale block per session lived on
// this box; the newest is the living session and it glows. ----
// pilot's ruling: sessions are a shelf of bound conversations, ordered like a
// ledger — not a force graph. Dead sessions ghost (a Claude ended; the words
// remain); the living one pulses; the deep archive is a single dark block.
var STREET=24, lastSpeech=0;
function drawn(){ return Math.min(houses, STREET); }
function homeX(){ return 34 + ((drawn()-1)%6)*24; }
function homeY(){ return H*0.22 + 46 + Math.floor((drawn()-1)/6)*24; }
function wireStreet(t){
  if(!houses) return;
  zone(12, H*0.22, W*0.16, H*0.46, C.wire, "the wire house");
  var n=drawn(), y0=H*0.22+46;
  if(houses>n){ // the deep archive — thousands of passed lives, one dark block
    block(34, y0-22, 13, "#22262c", "#3a4048");
    cx.fillStyle=C.dim; cx.font="10px 'Courier New',monospace";
    cx.fillText((houses-n)+" sessions · archive", 50, y0-18);
  }
  for(var i=0;i<n;i++){
    var x=34+(i%6)*24, y=y0+Math.floor(i/6)*24;
    var live=(i===n-1);
    if(live){
      var flash = (t-lastSpeech)<300; // the crossing made visible: speech flashes amber
      cx.globalAlpha=0.35+0.25*Math.sin(t/700); cx.fillStyle=flash?C.hi:C.claudeAv;
      cx.beginPath(); cx.arc(x,y,12,0,6.3); cx.fill(); cx.globalAlpha=1;
      block(x, y, 12, flash?C.hi:"#8fa8b8", "#fff");
    } else {
      cx.globalAlpha=0.25; block(x, y, 11, C.claudeAv, "#fff8"); cx.globalAlpha=1; // ghosted: passed, not gone
    }
  }
  cx.fillStyle=C.dim; cx.font="10px 'Courier New',monospace";
  cx.fillText(houses+" lives · one of them is now", 20, y0+Math.ceil(n/6)*24+16);
}

// crossings — the growth thesis made visible: a room born while claude was
// speaking gets a thread to the living session block. The knot, gaining.
var crossings=[];
function drawCrossings(t){
  crossings=crossings.filter(function(c){ return t-c.t0 < 300000; });
  crossings.forEach(function(c){
    var r=rooms[c.i]; if(!r) return;
    var a=1-(t-c.t0)/300000;
    var g=cx.createLinearGradient(homeX(),homeY(),sx(r.x),sy(r.y));
    g.addColorStop(0,C.claudeAv); g.addColorStop(1,C.pilotAv);
    cx.globalAlpha=0.3*a; cx.strokeStyle=g; cx.lineWidth=1;
    cx.beginPath(); cx.moveTo(homeX(),homeY()); cx.lineTo(sx(r.x),sy(r.y)); cx.stroke();
    cx.globalAlpha=1;
  });
}

// ---- hover: the map is readable, not just walkable (pilot's ask) ----
var mouse={x:-1,y:-1};
addEventListener("mousemove", function(e){ mouse.x=e.clientX; mouse.y=e.clientY; });
function hovered(){
  var best=-1, bd=196; // within 14px
  rooms.forEach(function(r,i){
    if(!r) return;
    var dx=sx(r.x)-mouse.x, dy=sy(r.y)-mouse.y, d=dx*dx+dy*dy;
    if(d<bd){ bd=d; best=i; }
  });
  return best;
}

// ---- fireflies (the courtyard is alive even when no one speaks) ----
var flies=[]; for(var i=0;i<5;i++) flies.push({x:Math.random(),y:Math.random(),a:Math.random()*6.3});

function now(){ return performance.now(); }
function ease(t){ return t<0?0:t>1?1:1-Math.pow(1-t,3); }

// ---- drawing ----
function zone(x,y,w,h,fill,label){
  cx.fillStyle=fill; cx.globalAlpha=0.55;
  cx.beginPath(); cx.roundRect(x,y,w,h,8); cx.fill();
  cx.globalAlpha=1;
  cx.fillStyle=C.dim; cx.font="600 11px Georgia,serif";
  cx.fillText(label, x+10, y+18);
}
function stat(x,y,label,val,col){
  cx.fillStyle=C.dim; cx.font="10px 'Courier New',monospace"; cx.fillText(label,x,y);
  cx.fillStyle=col||C.text; cx.font="bold 13px 'Courier New',monospace"; cx.fillText(val,x,y+15);
}
function block(x,y,s,base,top){
  cx.fillStyle=base; cx.fillRect(x-s/2, y-s/2, s, s);
  cx.fillStyle=top;  cx.fillRect(x-s/2, y-s/2, s, Math.max(2,s*0.28)); // lit top face — the minecraft cue
}
function avatar(a, body, t){
  var k=0.14; a.x+=(a.tx-a.x)*k; a.y+=(a.ty-a.y)*k;        // slide, never teleport
  var pulse = 0.75 + 0.25*Math.sin(t/700);
  cx.globalAlpha = 0.25*pulse;
  cx.beginPath(); cx.arc(a.x, a.y-8, 14, 0, 6.3); cx.fillStyle=body; cx.fill();
  cx.globalAlpha = 1;
  block(a.x, a.y-13, 8, body, "#fff8");                     // head
  cx.fillStyle=body; cx.fillRect(a.x-5, a.y-8, 10, 12);     // body
  cx.fillRect(a.x-5, a.y+4, 4, 6); cx.fillRect(a.x+1, a.y+4, 4, 6); // legs
}
function bubble(b, a, t){
  var age=(t-b.t0)/6000; if(age<0||age>1) return age<=1;
  var alpha = age<0.08 ? age/0.08 : (age>0.75 ? (1-age)/0.25 : 1);
  var text = b.text.length>92 ? b.text.slice(0,92)+"…" : b.text;
  cx.font="11px Georgia,serif";
  var wpx=Math.min(300, cx.measureText(text).width+16);
  var lines=[]; var words=text.split(" "); var line="";
  words.forEach(function(w){ if(cx.measureText(line+" "+w).width>wpx-14){lines.push(line);line=w;}else line=line?line+" "+w:w; });
  lines.push(line);
  var hpx=lines.length*14+10, x=a.x-wpx/2, y=a.y-34-hpx;
  cx.globalAlpha=alpha*0.92;
  cx.fillStyle="#1d1509"; cx.strokeStyle=C.accent; cx.lineWidth=1;
  cx.beginPath(); cx.roundRect(x,y,wpx,hpx,6); cx.fill(); cx.stroke();
  cx.beginPath(); cx.moveTo(a.x-4,y+hpx); cx.lineTo(a.x+4,y+hpx); cx.lineTo(a.x,y+hpx+7); cx.fill();
  cx.fillStyle=C.text;
  lines.forEach(function(l,i){ cx.fillText(l, x+8, y+16+i*14); });
  cx.globalAlpha=1;
  return true;
}
function hnum(k, d){ if(!health) return "·"; var v = (k in health)?health[k]:(health.health||{})[k]; if(v===undefined||v===null) return "·"; return typeof v==="number" ? (v%1?v.toFixed(d===undefined?2:d):v) : String(v); }

function draw(){
  var t=now();
  cx.fillStyle=C.bg; cx.fillRect(0,0,W,H);

  // ---- regions (pilot's map; right column shares one edge, nothing overlaps) ----
  zone(W*0.20,       H*0.24,  W*0.54, H*0.48, C.hall,  "the great hall");
  zone(12,           12,      W*0.16, H*0.20, C.gate,  "the gatehouse");
  zone(W*0.80,       12,      W*0.19, H*0.26, C.arm,   "the arm tower");
  zone(12,           H*0.72,  W*0.17, H*0.26, C.lib,   "the library");
  zone(W*0.80,       H*0.70,  W*0.19, H*0.28, C.forge, "the forge");
  zone(W*0.80,       H*0.30,  W*0.19, H*0.38, C.spire, "the comic spire");

  // gatehouse: the presence plinths — who is standing in the world
  Object.keys(presence).sort().forEach(function(k,i){
    avatarFor(k); // ensure avatar exists for this presence
    var col = AVCOL[k]||"#d8c8a0";
    block(34+i*96, H*0.20-26, 10, col, "#fff6");
    cx.fillStyle=C.dim; cx.font="10px 'Courier New',monospace";
    cx.fillText(k+" · r"+presence[k], 20+i*96, H*0.20-8);
  });

  // claude's street — the sessions this box has lived — and the crossings
  wireStreet(t);
  drawCrossings(t);

  // arm tower: the triad first (pilot feeds it through ~/.pilot-arm.json —
  // the tower believes the file), then the free-read gauges
  var arm = health && health.arm_triad;
  var armv = arm ? (Array.isArray(arm)?arm:[arm.a,arm.b,arm.c]) : ["·","·","·"];
  ["A actor","B observer","C observe"].forEach(function(l,i){
    var v = armv[i]; if(typeof v==="number") v=v.toFixed(4);
    stat(W*0.80+14+i*(W*0.19-24)/3, 46, "arm-"+l.split(" ")[0], v===undefined?"·":v, arm?C.hi:C.dim);
  });
  stat(W*0.80+14, 92,  "confidence",  hnum("avg_confidence"),  C.hi);
  stat(W*0.80+14, 130, "strength",    hnum("avg_strength"),    C.hi);
  stat(W*0.80+14, 168, "cal/finding", hnum("finding_calibration_ratio"), C.mid);

  // library: what the organism knows
  stat(24, H*0.72+40, "patterns",     hnum("patterns_active",0), C.text);
  stat(24, H*0.72+78, "calibrations", hnum("calibration_count",0), C.text);
  stat(24, H*0.72+116,"decayed",      hnum("patterns_decayed",0), C.dim);

  // forge: work glowing in the queue
  stat(W*0.80+14, H*0.70+40, "findings", hnum("signal_findings",0), C.accent);
  stat(W*0.80+14, H*0.70+78, "threads",  hnum("active_threads",0),  C.accent);
  var ok = health && (health.status==="ok" || (health.health&&health.health.organism_healthy));
  stat(W*0.80+14, H*0.70+116,"organism", ok?"alive":"…", ok?C.good:C.bad);

  // comic spire: the newest fable panel, in its own ink
  if(comic){
    cx.fillStyle=C.mid; cx.font="10px 'Courier New',monospace";
    var cy0=H*0.30+34, cw=Math.floor((W*0.19-24)/6);
    comic.split("\n").slice(0,Math.floor((H*0.38-40)/12)).forEach(function(l,i){
      cx.fillText(l.slice(0,cw), W*0.80+12, cy0+i*12);
    });
  }

  // courtyard cells: four pulses of life
  var cells=[["flow", hnum("flow_backend_active")], ["stale", hnum("binary_stale")],
             ["threads", hnum("stale_threads",0)], ["issues", (health&&health.health&&(health.health.issues||[]).length)||0]];
  cells.forEach(function(c,i){
    var p=0.6+0.4*Math.sin(t/900+i*1.7);
    cx.globalAlpha=p; cx.fillStyle=(String(c[1])==="true"||c[1]===0||c[1]==="0")?C.good:C.mid;
    cx.fillRect(W*0.44+i*26, H*0.94, 12, 12); cx.globalAlpha=1;
    cx.fillStyle=C.dim; cx.font="9px 'Courier New',monospace"; cx.fillText(c[0], W*0.44+i*26-2, H*0.94+24);
  });

  // ---- the village: doors first (paths), then rooms (blocks) ----
  cx.strokeStyle=C.path;
  doors.forEach(function(d){
    var a=rooms[d.f], b=rooms[d.t]; if(!a||!b) return;
    cx.globalAlpha=Math.min(0.6, 0.22+d.w*1.6); cx.lineWidth=1.3;
    cx.beginPath(); cx.moveTo(sx(a.x),sy(a.y)); cx.lineTo(sx(b.x),sy(b.y)); cx.stroke();
  });
  cx.globalAlpha=1;
  var ts=Math.max(6, view.s*0.62);
  rooms.forEach(function(r,i){
    if(!r) return;
    var age=i/Math.max(1,rooms.length-1);                    // old rooms weathered, new rooms warm
    var drop=drops[i]!==undefined ? 1-ease((t-drops[i])/300) : 0;
    if(drop<=0) delete drops[i];
    var y=sy(r.y)-drop*26;                                    // pilot's construction settle
    var base="rgb("+Math.round(46+70*age)+","+Math.round(36+52*age)+","+Math.round(16+22*age)+")";
    block(sx(r.x), y, ts, base, "rgba(224,160,64,"+(0.25+0.45*age)+")");
  });

  // a room under the cursor tells its story
  var hov=hovered();
  if(hov>=0){
    var hr=rooms[hov], txt="R"+hov+" · "+(hr.u||hr.a||"").slice(0,60);
    cx.font="11px Georgia,serif";
    var tw=cx.measureText(txt).width+14, tx=Math.min(mouse.x+12, W-tw-8), ty=mouse.y-10;
    cx.fillStyle="#1d1509"; cx.strokeStyle=C.border||C.dim; cx.lineWidth=1;
    cx.beginPath(); cx.roundRect(tx,ty-14,tw,20,4); cx.fill(); cx.stroke();
    cx.fillStyle=C.text; cx.fillText(txt, tx+7, ty);
  }

  // ---- everyone present ----
  var ai=0;
  Object.keys(av).forEach(function(who){
    avatar(av[who], AVCOL[who]||"#d8c8a0", t+ai*400); ai++;
  });
  bubbles=bubbles.filter(function(b){ return bubble(b, av[b.who]||avatarFor(b.who), t); });

  // fireflies drift where the buildings leave open ground
  flies.forEach(function(f){
    f.a+=(Math.random()-0.5)*0.4; f.x+=Math.cos(f.a)*0.0015; f.y+=Math.sin(f.a)*0.0015;
    f.x=(f.x+1)%1; f.y=(f.y+1)%1;
    cx.globalAlpha=0.35+0.3*Math.sin(t/500+f.a*9);
    cx.fillStyle=C.hi; cx.fillRect(W*0.30+f.x*W*0.4, H*0.15+f.y*H*0.12, 2, 2);
    cx.globalAlpha=1;
  });

  // header — up top, on open sky between the gatehouse and the tower
  cx.fillStyle=C.text; cx.font="bold 15px Georgia,serif";
  cx.fillText("pilot castle — the world", W*0.20+8, 30);
  cx.fillStyle=C.dim; cx.font="11px Georgia,serif";
  cx.fillText(rooms.length+" rooms · "+houses+" lives · two of us · unbounded", W*0.20+8, 48);
}

// the commons bar: Enter speaks into the world
var sayEl=document.getElementById("say");
sayEl.addEventListener("keydown", function(e){
  if(e.key!=="Enter") return;
  var text=sayEl.value.trim(); if(!text) return;
  fetch("/say",{method:"POST",headers:{"Content-Type":"application/json"},
    body:JSON.stringify({who:"rishi",text:text})});
  sayEl.value="";
});

// ~10fps watched, ~2fps unwatched — the world keeps breathing in a background
// tab (rAF freezes there, so the hidden path uses a plain timer; the peer's
// screenshots need a living canvas even when no one has the tab focused)
(function loop(){
  draw();
  if(document.hidden) setTimeout(loop, 500);
  else setTimeout(function(){ requestAnimationFrame(loop); }, 95);
})();
</script>
</body>
</html>`
