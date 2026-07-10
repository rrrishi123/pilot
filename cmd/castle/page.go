package main

// The castle, as an actual game. A 2D side-view sandbox — four finite verbs
// (walk, jump, break, place) over a tile world, which is the whole point: the
// mechanics are bounded, what agents BUILD with them is not. This is the
// opposite of the old passive canvas that generated visuals from observations.
//
// The memory-rooms don't vanish — they become WOOD structures standing on the
// terrain: walk into them, mine them, build over them. Block edits POST to
// /edit (persisted + broadcast) so the world is shared and durable — the
// "file, believed" contract, now for a world you can dig. Other players are
// real avatars at real positions (POST /pos), not decorative presence dots.
//
// Engine: fixed-timestep update(dt) + render(); tile AABB collision resolved
// X-then-Y; camera follows the player; only visible tiles are drawn.
const page = `<!DOCTYPE html>
<html lang="en"><head><meta charset="UTF-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>pilot castle · the world</title>
<style>
  html,body{margin:0;height:100%;background:#0b0d12;overflow:hidden;font-family:Georgia,serif}
  #c{display:block;width:100vw;height:100vh;image-rendering:pixelated;cursor:crosshair}
  #hud{position:fixed;left:10px;top:8px;color:#d4b464;font:12px 'Courier New',monospace;text-shadow:0 1px 2px #000;pointer-events:none;line-height:1.5}
  #hot{position:fixed;left:50%;bottom:44px;transform:translateX(-50%);display:flex;gap:6px;z-index:8}
  .slot{width:34px;height:34px;border:2px solid #3a2810;background:#181209;display:flex;align-items:center;justify-content:center;font:10px monospace;color:#a88040}
  .slot.on{border-color:#e0a040;box-shadow:0 0 8px #e0a04066}
  #minimap{position:fixed;left:8px;bottom:86px;border:2px solid #3a2810;border-radius:3px;image-rendering:pixelated;cursor:pointer;z-index:10;width:200px;height:110px}
  #saybar{position:fixed;left:8px;bottom:8px;right:8px;display:flex;gap:6px;z-index:5}
  #saybar input{flex:1;background:#12141a;border:1px solid #3a2810;color:#d4b464;font:12px 'Courier New',monospace;padding:4px 8px;border-radius:2px;outline:none}
  #saybar input:focus{border-color:#c07828}
  #saybar button{background:#1a1510;border:1px solid #3a2810;color:#a88040;font:11px monospace;padding:4px 12px;cursor:pointer;border-radius:2px}
  #saybar button:hover{background:#2a2015;color:#e0a040}
  #gopher-mirror{position:fixed;right:0;top:0;bottom:0;width:280px;background:rgba(11,13,18,0.92);border-left:1px solid #3a2810;z-index:20;display:flex;flex-direction:column;font:10px 'Courier New',monospace;overflow:hidden}
  #gopher-mirror .header{padding:6px 8px;background:#1a1510;border-bottom:1px solid #3a2810;color:#d4b464;font-weight:bold;font-size:11px;display:flex;gap:8px}
  #gopher-mirror .header span{cursor:pointer;opacity:0.6}
  #gopher-mirror .header span.on{opacity:1;color:#e0a040}
  #gopher-mirror .col{flex:1;overflow-y:auto;padding:4px 6px;display:none}
  #gopher-mirror .col.on{display:block}
  #gopher-mirror .col::-webkit-scrollbar{width:3px}
  #gopher-mirror .col::-webkit-scrollbar-thumb{background:#3a2810}
  .gmsg{margin:2px 0;padding:2px 4px;border-left:2px solid #3a2810;line-height:1.4}
  .gmsg.think{border-color:#5a7a8a;color:#7a9aaa}
  .gmsg.act{border-color:#c07828;color:#c8a060}
  .gmsg.result{border-color:#4a7a3a;color:#6aaa50}
  .gmsg.speak{border-color:#a88040;color:#d4b464}
  .gmsg.whisper{border-color:#8060a0;color:#a080c0}
  .gmsg .gt{color:rgba(255,255,255,0.35);font-size:8px}
  #gm-toggle{position:fixed;right:8px;top:8px;z-index:25;cursor:pointer;font:16px monospace;color:#a88040;background:#1a1510;border:1px solid #3a2810;border-radius:3px;padding:2px 6px;opacity:0.7}
  #gm-toggle:hover{opacity:1}
</style></head>
<body>
<canvas id="c"></canvas>
<div id="hud"></div>
<div id="hot"></div>
<canvas id="minimap" width="200" height="110"></canvas>
<div id="saybar"><input id="say" placeholder="speak… /go gophers, /locate pilot-a, SHIFT-click minimap to teleport" autocomplete="off"><button onclick="document.getElementById('say').dispatchEvent(new KeyboardEvent('keydown',{key:'Enter'}))">say</button></div>
<script>
"use strict";
(function hb(){var lt=0;setInterval(function(){var n=performance.now(),dt=Math.min(0.05,(n-lt)/1000);lt=n;try{if(typeof update==="function"){update(dt);cam.x=Math.round((player.x+player.w/2-W/2));cam.y=Math.round((player.y+player.h/2-H/2));}if(typeof render==="function")render(n);if(typeof hud==="function")try{hud();}catch(e){}}catch(e){}},80);})();
var cv=document.getElementById("c"), cx=cv.getContext("2d");
var W=0,H=0,DPR=Math.min(devicePixelRatio||1,2);
function resize(){W=innerWidth;H=innerHeight;cv.width=W*DPR;cv.height=H*DPR;cx.setTransform(DPR,0,0,DPR,0,0);cx.imageSmoothingEnabled=false;}
resize();addEventListener("resize",resize);

var TS=30;                          // tile size (px)
var AIR=0,GRASS=1,DIRT=2,STONE=3,WOOD=4,PLANK=5,LEAF=6;
var COL={1:"#4c7a3a",2:"#6b4a2b",3:"#5a5a62",4:"#7a4a24",5:"#c89a5a",6:"#3a6a34"};
var TOP={1:"#63a04a",3:"#6f6f78",4:"#8f5a2c",5:"#e0b878"};   // lit top face
var SOLID={1:1,2:1,3:1,4:1,5:1};    // LEAF(6)+AIR(0) are passable
var world={};                       // "tx,ty" -> tile type (only non-terrain overrides live here)
var rooms=[], seed=1337, WW=260, GH=42; // world width in tiles, ground row
var me = "claude-"+Math.floor((performance.now()*7)%9999);
var others={};                      // who -> {x,y,tx,ty,col,t}
var bubbles=[];                     // {who,text,t0} — speech bubbles that float up & pop

// ---- deterministic terrain (seeded; same world for every player) ----
function rnd(n){ n=(n*1103515245+12345+seed)&0x7fffffff; return ((n>>16)&0x7fff)/0x7fff; }
function surfaceH(tx){ // rolling hills
  return Math.floor(GH + 4*Math.sin(tx*0.14+seed) + 2.5*Math.sin(tx*0.4) + 2*rnd(tx*13)-1);
}
function terrain(tx,ty){ // base tile before edits
  var s=surfaceH(tx);
  if(ty<s) return AIR;
  if(ty===s) return GRASS;
  if(ty<s+4) return DIRT;
  return STONE;
}
function tileAt(tx,ty){ var k=tx+","+ty; if(k in world) return world[k]; return terrain(tx,ty); }
function setTile(tx,ty,v){ world[tx+","+ty]=v; }

// rooms become wood houses standing on the surface, spread across the world
function buildRooms(){
  rooms.forEach(function(r,i){
    var bx=8+i*7, s=surfaceH(bx), w=4, h=3;                 // a little 4x3 hut + doorway
    r.bx=bx; r.by=s-h;
    for(var x=bx;x<bx+w;x++) for(var y=s-h;y<s;y++){
      var edge=(x===bx||x===bx+w-1||y===s-h);
      var door=(x===bx+1&&(y===s-1||y===s-2));
      if(edge&&!door) setTile(x,y,WOOD); else if(!(x+","+y in world)) setTile(x,y,AIR);
    }
    for(var x2=bx-1;x2<=bx+w;x2++) setTile(x2,s-h-1,LEAF); // a leafy roof line
  });
}

// ---- data + live sync ----
// loadWorld is re-callable: at boot, on SSE reconnect, and on fetch failure (retry).
// A tab that loads while the castle is restarting must not stay half-born forever.
var worldLoaded=false;
function loadWorld(){
  fetch("/world.json").then(function(r){ if(!r.ok) throw new Error("castle "+r.status); return r.json();}).then(function(d){
    rooms=d.rooms||[]; if(typeof d.seed==="number") seed=d.seed; if(d.rooms)for(var ri=0;ri<d.rooms.length;ri++)rooms[ri].n=d.rooms[ri].n;
    (d.edits||[]).forEach(function(e){ world[e.x+","+e.y]=e.b; });
    buildRooms();
    // seed agent positions into others so the minimap shows everyone immediately
    if(d.agentPos) for(var w in d.agentPos){ var a=d.agentPos[w]; if(w!==me){ others[w]=others[w]||{x:a.x,y:a.y,tx:a.x,ty:a.y,t:performance.now()}; } }
    if(!worldLoaded){ var sx3=surfaceH(-30); player.x=-30*TS; player.y=(sx3-3)*TS; } // spawn only on first load
    worldLoaded=true;
  }).catch(function(){ setTimeout(loadWorld, 3000); }); // castle down/restarting — keep trying
}
loadWorld();
var es=new EventSource("/events");
var esWasDown=false;
es.onerror=function(){ esWasDown=true; };
es.onopen=function(){ if(esWasDown){ esWasDown=false; loadWorld(); } }; // reconnected: refetch missed state
es.onmessage=function(m){ var e; try{e=JSON.parse(m.data);}catch(_){return;}
  if(e.type==="edit"){ world[e.x+","+e.y]=e.b; }
  if(e.type==="pos" && e.who!==me){ var o=others[e.who]||(others[e.who]={x:e.x,y:e.y}); o.tx=e.x; o.ty=e.y; o.t=performance.now(); }
  if(e.type==="speech"){ bubbles.push({who:e.who,text:e.text,t0:performance.now()}); }
};
function post(url,obj){ fetch(url,{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify(obj)}).catch(function(){}); }

// ---- player ----
var player={x:WW/2*TS,y:GH*TS,w:20,h:44,vx:0,vy:0,onGround:false};
(function(){ var sx2=surfaceH(-30); player.x=-30*TS; player.y=(sx2-3)*TS; player.vx=0; player.vy=0; player.onGround=true; })(); // spawn near gopher territory
var keys={};
addEventListener("keydown",function(e){keys[e.key.toLowerCase()]=1; if([" ","arrowup","w","a","d","arrowleft","arrowright"].indexOf(e.key.toLowerCase())>=0)e.preventDefault();});
var moveTarget=null; // {tx,ty} to walk toward, or null  (click-to-move)
var moveTargetDot=0; // animation timer for the target indicator
addEventListener("keyup",function(e){keys[e.key.toLowerCase()]=0;});

var GRAV=1800, MOVE=2600, MAXVX=260, JUMP=560, FRICT=0.80;
function solidAt(px,py){ return !!SOLID[tileAt(Math.floor(px/TS),Math.floor(py/TS))]; }
function hitsWorld(x,y,w,h){ // AABB vs solid tiles
  var x0=Math.floor(x/TS),x1=Math.floor((x+w-1)/TS),y0=Math.floor(y/TS),y1=Math.floor((y+h-1)/TS);
  for(var tx=x0;tx<=x1;tx++)for(var ty=y0;ty<=y1;ty++) if(SOLID[tileAt(tx,ty)]) return true;
  return false;
}
function update(dt){
  // click-to-move: auto-walk toward the target tile
  if(moveTarget && !keys["a"]&&!keys["d"]&&!keys["arrowleft"]&&!keys["arrowright"]){
    var tgtPixel=moveTarget.tx*TS+TS/2; // pixel center of target tile
    var dx=tgtPixel-player.x;
    if(Math.abs(dx)>8) player.vx+=(dx>0?MOVE:-MOVE)*dt;
    else{ player.vx=0; moveTarget=null; } // arrived
    // auto-jump if blocked
    if(Math.abs(dx)<50 && player.onGround && hitsWorld(player.x+player.vx*dt*2,player.y,player.w,player.h)){
      player.vy=-JUMP; player.onGround=false;
    }
  } else {
    var ax=0;
    if(keys["a"]||keys["arrowleft"]) ax-=MOVE;
    if(keys["d"]||keys["arrowright"]) ax+=MOVE;
    player.vx+=ax*dt; if(ax===0) player.vx*=FRICT;
    player.vx=Math.max(-MAXVX,Math.min(MAXVX,player.vx));
  }
  if((keys["w"]||keys[" "]||keys["arrowup"])&&player.onGround){ player.vy=-JUMP; player.onGround=false; }
  player.vy+=GRAV*dt; if(player.vy>900)player.vy=900;
  // move X, resolve
  var nx=player.x+player.vx*dt;
  if(!hitsWorld(nx,player.y,player.w,player.h)) player.x=nx; else { player.vx=0; }
  // move Y, resolve
  var ny=player.y+player.vy*dt;
  if(!hitsWorld(player.x,ny,player.w,player.h)){ player.y=ny; player.onGround=false; }
  else { if(player.vy>0) player.onGround=true; player.vy=0; }
  if(player.y>WW*2*TS){ var s=surfaceH(Math.floor(player.x/TS)); player.y=(s-3)*TS; player.vy=0; } // fell out -> respawn on surface
  // cancel moveTarget when any movement key pressed
  if(keys["a"]||keys["d"]||keys["arrowleft"]||keys["arrowright"]||keys["w"]||keys[" "]) moveTarget=null;
}
// broadcast my position (throttled)
var lastPos=0;
function syncPos(t){ if(t-lastPos>150){ lastPos=t; post("/pos",{who:me,x:Math.round(player.x),y:Math.round(player.y)}); } }

// ---- break / place ----
var mouse={x:0,y:0,down:0,btn:0};
cv.addEventListener("contextmenu",function(e){e.preventDefault();});
cv.addEventListener("mousemove",function(e){mouse.x=e.clientX;mouse.y=e.clientY;});
cv.addEventListener("mousedown",function(e){ mouse.down=1; mouse.btn=e.button; act(); e.preventDefault(); });
cv.addEventListener("mouseup",function(){mouse.down=0;});
var hotbar=[STONE,PLANK,WOOD,DIRT], sel=0;
addEventListener("keydown",function(e){ var n=parseInt(e.key); if(n>=1&&n<=hotbar.length) sel=n-1; });
function targetTile(){ var wx=mouse.x+cam.x, wy=mouse.y+cam.y; return {tx:Math.floor(wx/TS),ty:Math.floor(wy/TS)}; }
function reachOK(tx,ty){ var cxp=(player.x+player.w/2)/TS, cyp=(player.y+player.h/2)/TS; return Math.hypot(tx+0.5-cxp,ty+0.5-cyp)<6; }
function act(){
  var t=targetTile(); if(!reachOK(t.tx,t.ty)) return;
  if(mouse.btn===0){ // left-click: break OR walk
    if(SOLID[tileAt(t.tx,t.ty)]){ setTile(t.tx,t.ty,AIR); post("/edit",{who:me,x:t.tx,y:t.ty,b:AIR}); moveTarget=null; }
    else { moveTarget={tx:t.tx,ty:t.ty}; } // walk to the clicked tile
  } else if(mouse.btn===2){ // right-click: place OR walk
    if(tileAt(t.tx,t.ty)===AIR){
      var pl={x:player.x,y:player.y,w:player.w,h:player.h}, bx=t.tx*TS,by=t.ty*TS;
      if(!(bx<pl.x+pl.w&&bx+TS>pl.x&&by<pl.y+pl.h&&by+TS>pl.y)){ setTile(t.tx,t.ty,hotbar[sel]); post("/edit",{who:me,x:t.tx,y:t.ty,b:hotbar[sel]}); moveTarget=null; }
      else { moveTarget={tx:t.tx,ty:t.ty}; }
    } else { moveTarget={tx:t.tx,ty:t.ty}; } // click on solid = walk there
  }
}

// ---- camera + render ----
var cam={x:0,y:0};
function drawTile(tx,ty,type){
  var x=tx*TS-cam.x, y=ty*TS-cam.y;
  cx.fillStyle=COL[type]||"#222"; cx.fillRect(x,y,TS,TS);
  if(TOP[type] && tileAt(tx,ty-1)===AIR){ cx.fillStyle=TOP[type]; cx.fillRect(x,y,TS,Math.max(3,TS*0.22)); }
  cx.strokeStyle="rgba(0,0,0,0.18)"; cx.strokeRect(x+0.5,y+0.5,TS,TS);
}
function drawGuy(px,py,col,label){
  var x=px-cam.x, y=py-cam.y;
  cx.fillStyle=col; cx.fillRect(x,y,player.w,player.h);
  cx.fillStyle="#fff8"; cx.fillRect(x,y,player.w,6);          // lit head
  if(label){ cx.fillStyle="#d4b464"; cx.font="10px 'Courier New',monospace"; cx.textAlign="center"; cx.fillText(label,x+player.w/2,y-4); cx.textAlign="left"; }
}
function render(t){
  cam.x=Math.round((player.x+player.w/2-W/2)); cam.y=Math.round((player.y+player.h/2-H/2));
  // sky gradient
  var g=cx.createLinearGradient(0,0,0,H); g.addColorStop(0,"#0b0d12"); g.addColorStop(1,"#1a2230"); cx.fillStyle=g; cx.fillRect(0,0,W,H);
  // visible tiles
  var tx0=Math.floor(cam.x/TS)-1, tx1=Math.floor((cam.x+W)/TS)+1, ty0=Math.floor(cam.y/TS)-1, ty1=Math.floor((cam.y+H)/TS)+1;
  for(var tx=tx0;tx<=tx1;tx++)for(var ty=ty0;ty<=ty1;ty++){ var v=tileAt(tx,ty); if(v!==AIR) drawTile(tx,ty,v); }
  // room labels above their huts
  cx.font="11px Georgia,serif"; cx.textAlign="center";
  rooms.forEach(function(r){ if(r.bx==null)return; var x=(r.bx+2)*TS-cam.x, y=(r.by-1)*TS-cam.y; if(x>-80&&x<W+80){ cx.fillStyle="#a88040"; cx.font="11px Georgia,serif"; cx.textAlign="center"; cx.fillText("R"+rooms.indexOf(r),x,y); var by=r.n||"unknown"; cx.font="8px monospace"; cx.fillStyle="rgba(168,128,64,0.6)"; cx.fillText(by,x,y+12); cx.textAlign="left"; } });
  cx.textAlign="left";
  // other players
  var now=performance.now();
  for(var w in others){ var o=others[w]; if(now-o.t>8000){delete others[w];continue;} o.x+=((o.tx||o.x)-o.x)*0.2; o.y+=((o.ty||o.y)-o.y)*0.2; drawGuy(o.x,o.y,"#e0a040",w.slice(0,10)); }
  // me
  drawGuy(player.x,player.y,"#b8ccd8",me.slice(0,10));
  // target highlight
  var tt=targetTile(); if(reachOK(tt.tx,tt.ty)){ cx.strokeStyle=SOLID[tileAt(tt.tx,tt.ty)]?"#e0a040":"#7a9a50"; cx.lineWidth=2; cx.strokeRect(tt.tx*TS-cam.x,tt.ty*TS-cam.y,TS,TS); cx.lineWidth=1; }
  if(moveTarget){
    var mx=moveTarget.tx*TS+TS/2-cam.x, my=(moveTarget.ty+1)*TS-cam.y;
    var pulse=0.4+0.3*Math.sin(performance.now()/1500);
    cx.strokeStyle="rgba(255,220,100,"+pulse+")"; cx.lineWidth=1.5;
    cx.setLineDash([4,6]); cx.beginPath();
    cx.moveTo(mx-8,my); cx.lineTo(mx+8,my);
    cx.moveTo(mx,my-8); cx.lineTo(mx,my+8);
    cx.stroke(); cx.setLineDash([]); cx.lineWidth=1;
  }
  // civilisation layer (zones, homes, locate rings)
  if(typeof renderCivilisation==="function") renderCivilisation(t);
  // minimap overlay
  if(typeof drawMinimap==="function") drawMinimap(t);
  // ---- speech bubbles: pop up from head, float upward, fade out ----
  var BUBBLE_DUR=5000; // ms total lifetime
  var FLOAT_PX=50;     // total upward float distance over lifetime
  bubbles=bubbles.filter(function(b){
    var age=(now-b.t0)/BUBBLE_DUR;
    if(age>1) return false; // expired — remove
    // Find who spoke (player self or others mapping)
    var who=b.who===me ? player : (others[b.who]);
    if(!who) return true; // keep waiting if not in view yet
    // Position: above the character's head
    var bx=(who.x||player.x)-cam.x + 10;   // center offset
    var by=(who.y||player.y)-cam.y - 24;   // base y: above head
    // Float upward: age 0→1 pushes bubble up by FLOAT_PX
    var rise = age * FLOAT_PX;
    by -= rise;
    // Fade: starts fading at 60%, fully gone by 100%
    var alpha = age<0.6 ? 1 : (1-age)/0.4;
    if(alpha<=0) return false;
    // Scale pop-in at birth: age 0→0.15 grows from tiny to full
    var popScale = age<0.15 ? (age/0.15) : 1;
    cx.save();
    cx.globalAlpha=Math.max(0,Math.min(1,alpha));
    // Round-rect bubble background
    var txt=b.text.slice(0,60);
    var tw=cx.measureText(txt).width+14;
    var bh=20;
    var bw=tw;
    var bbx=bx-bw/2;
    var bby=by-bh;
    var r=6; // corner radius
    // Draw rounded rectangle
    cx.beginPath();
    cx.moveTo(bbx+r,bby);
    cx.lineTo(bbx+bw-r,bby);
    cx.quadraticCurveTo(bbx+bw,bby,bbx+bw,bby+r);
    cx.lineTo(bbx+bw,bby+bh-r);
    cx.quadraticCurveTo(bbx+bw,bby+bh,bbx+bw-r,bby+bh);
    cx.lineTo(bbx+r,bby+bh);
    cx.quadraticCurveTo(bbx,bby+bh,bbx,bby+bh-r);
    cx.lineTo(bbx,bby+r);
    cx.quadraticCurveTo(bbx,bby,bbx+r,bby);
    cx.closePath();
    cx.fillStyle="rgba(18,14,8,0.85)";
    cx.fill();
    cx.strokeStyle="rgba(192,120,40,0.6)";
    cx.lineWidth=1;
    cx.stroke();
    // Tail: little triangle pointing down at the speaker
    var tailX=bx, tailY=bby+bh-1;
    cx.beginPath();
    cx.moveTo(tailX-5,tailY);
    cx.lineTo(tailX+5,tailY);
    cx.lineTo(tailX,tailY+6);
    cx.closePath();
    cx.fillStyle="rgba(18,14,8,0.85)";
    cx.fill();
    cx.strokeStyle="rgba(192,120,40,0.6)";
    cx.lineWidth=1;
    cx.stroke();
    // Text
    cx.fillStyle="#d4b464";
    cx.font="10px Georgia,serif";
    cx.textAlign="center";
    cx.textBaseline="middle";
    cx.fillText(txt,bx,by-bh/2);
    cx.textAlign="left";
    cx.textBaseline="alphabetic";
    cx.restore();
    return true;
  });
}

// ---- civilisation: homes, zones, search ----
var homes={}, zones=[], locateResult=null, locateT=0, cmdInput="";
var HOME_COL = {"pilot":"#e0a040","claude":"#b8ccd8","fable-pilot":"#c0a0e0","rishi":"#80c0a0"};
var NAME_MAP = {};
var sayEl=document.getElementById("say");
fetch("/homes").then(function(r){return r.json();}).then(function(h){ homes=h;
  for(var n in h){ var h2=h[n]; if(h2&&h2.tx!=null) NAME_MAP[h2.tx+","+h2.ty]=n; }
  for(var n in h){
    var t=h[n]; if(!t||t.tx==null) continue;
    var sx=surfaceH(t.tx);
    for(var x=t.tx-2;x<=t.tx+2;x++){ for(var y=sx-3;y<sx;y++){
      if(x===t.tx-2||x===t.tx+2||y===sx-3) setTile(x,y,STONE);
      else if(y===sx-1||y===sx-2) setTile(x,y,WOOD);
    }}
    setTile(t.tx,sx-4,PLANK); setTile(t.tx-1,sx-4,PLANK); setTile(t.tx+1,sx-4,PLANK);
  }
});
fetch("/zones").then(function(r){return r.json();}).then(function(z){ zones=z; });

// ---- command bar ----
sayEl.addEventListener("keydown", function(e){
  if(e.key!=="Enter") return;
  var text=sayEl.value.trim(); sayEl.value="";
  if(!text) return;
  var parts=text.split(" ");
  if(text[0]==="/"){
    var cmd=parts[0].toLowerCase();
    if(cmd==="/locate"||cmd==="/where"){
      var who=parts.slice(1).join(" ")||"pilot";
      fetch("/locate?who="+encodeURIComponent(who)).then(function(r){return r.json();}).then(function(d){
        locateResult=d; locateT=performance.now();
        var reply="";
        if(d.home) reply+=who+" home: ("+d.home.tx+","+d.home.ty+")";
        if(d.position) reply+=who+" at: ("+d.position.x+","+d.position.y+")";
        if(d.zone) reply+=who+" zone: "+d.zone;
        showMsg(reply);
        // Post result as speech so everyone sees it
        if(reply) fetch("/say",{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({who:me,text:reply})});
      });
      return;
    }
    if(cmd==="/go"||cmd==="/tp"){
      var target=parts[1];
      if(target==="gophers"||target==="all"){
        // Jump to center of all three gophers (commons -30,-8)
        var sx=surfaceH(-30);
        player.x=-30*TS; player.y=(sx-3)*TS; player.vx=0; player.vy=0;
        showMsg("jumped to gophers at commons (-30,-8)");
        return;
      }
      if(target==="home"||target==="spawn"){ var s=surfaceH(WW/2); player.x=WW/2*TS; player.y=(s-3)*TS; player.vx=0; player.vy=0; showMsg("returned to spawn"); }
      else if(homes[target]){ var h2=homes[target]; var sx=surfaceH(h2.tx); player.x=h2.tx*TS; player.y=(sx-3)*TS; player.vx=0; player.vy=0; showMsg("teleported to "+target); }
      else { showMsg("who is "+target+"?"); }
      return;
    }
    if(cmd==="/zones"||cmd==="/map"){ var info=""; zones.forEach(function(z){ info+=z.icon+" "+z.name+"  "; }); showMsg(info); return; }
    if(cmd==="/players"||cmd==="/who"){ var list=""; for(var w in others) list+=w+" "; showMsg("players: "+(Object.keys(others).length+1)+": "+list); return; }
    if(cmd==="/follow"){ var ft=parts[1]; if(others[ft]){ followTarget=ft; showMsg("following "+ft); } else showMsg("can't find "+ft); return; }
    if(cmd==="/unfollow"){ followTarget=null; showMsg("stopped"); return; }
  }
  fetch("/say",{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({who:me,text:text})});
});

var followTarget=null, msgText="", msgT=0;
function showMsg(t){ msgText=t; msgT=performance.now(); }
function renderCivilisation(t){

  zones.forEach(function(z){
    var x=z.tx*TS-cam.x, y=z.ty*TS-cam.y;
    if(x>-80&&x<W+80&&y>-40&&y<H+40){
      cx.globalAlpha=0.2+0.1*Math.sin(t/1200+z.tx*0.1);
      cx.fillStyle="#1d1509"; cx.fillRect(x-8,y-16,60,20);
      cx.fillStyle="#a88040"; cx.font="9px Georgia,serif"; cx.fillText(z.icon+" "+z.name, x-4, y-2);
      cx.globalAlpha=1;
    }
  });
  for(var n in homes){
    var h2=homes[n]; if(!h2||h2.tx==null) continue;
    var x=h2.tx*TS-cam.x, y=(surfaceH(h2.tx)-5)*TS-cam.y;
    if(x>-40&&x<W+40&&y>-20&&y<H+20){
      cx.fillStyle=HOME_COL[n]||"#d4b464";
      cx.font="bold 9px 'Courier New',monospace"; cx.textAlign="center";
      cx.fillText(n+"'s home", x+TS/2, y);
      cx.textAlign="left";
    }
  }
  if(locateResult&&t-locateT<8000&&locateResult.position){
    var lx=locateResult.position.x-cam.x, ly=locateResult.position.y-cam.y-40;
    if(lx>0&&lx<W&&ly>0&&ly<H){
      cx.strokeStyle="#e0a040"; cx.lineWidth=2; cx.setLineDash([4,4]);
      cx.beginPath(); cx.arc(lx,ly,24,0,6.3); cx.stroke(); cx.setLineDash([]);
      cx.fillStyle="#e0a040"; cx.font="bold 11px Georgia,serif"; cx.textAlign="center";
      cx.fillText(locateResult.name, lx, ly+36);
      cx.textAlign="left"; cx.lineWidth=1;
    }
  }
  if(followTarget&&others[followTarget]){
    var o=others[followTarget]; var dx=o.x-player.x;
    if(Math.abs(dx)>120) keys["d"]=dx>0?1:0, keys["a"]=dx<0?1:0;
    else keys["d"]=keys["a"]=0;
  }
}

// ---- HUD + hotbar ----
function hud(){
  try {
  var pcount=Object.keys(others).length+1;
  var zoneStr=""; var px=Math.floor((player.x+player.w/2)/TS), py=Math.floor((player.y+player.h/2)/TS);
  zones.forEach(function(z){ if(px>=z.tx&&px<z.tx+z.w&&py>=z.ty&&py<z.ty+z.h) zoneStr=z.icon+" "+z.name; });
  if(!zoneStr){ for(var n in homes){ var h2=homes[n]; if(h2&&Math.abs(px-h2.tx)<4&&Math.abs(py-h2.ty)<4){ zoneStr=n+"'s home"; break; }}}
  document.getElementById("hud").innerHTML="🏰 pilot castle — the world<br>walk: A/D · jump: W/Space · break: L-click · place: R-click · pick: 1-4<br>you: "+me+"  players: "+pcount+"  rooms: "+rooms.length+(zoneStr?" · "+zoneStr:"");
  if(msgText&&performance.now()-msgT<4000){
    document.getElementById("hud").innerHTML+="<br><span style='color:#e0a040'>"+msgText+"</span>";
  }
  } catch(e) { document.getElementById("hud").innerHTML="🏰 pilot castle — the world<br>walk: A/D · jump: W/Space · break: L-click · place: R-click · pick: 1-4 · rooms: "+(rooms?rooms.length:0); }
  var h=document.getElementById("hot"); if(h.childElementCount!==hotbar.length){ h.innerHTML=""; hotbar.forEach(function(b,i){ var d=document.createElement("div"); d.className="slot"; d.textContent=(typeof NAME!=='undefined'&&NAME[b])||'?'; h.appendChild(d); }); }
  [].forEach.call(h.children,function(el,i){ el.className="slot"+(i===sel?" on":""); });
}


// ---- minimap: birds-eye view of the whole civilisation ----
var mmCv=document.getElementById("minimap"), mmCx=mmCv.getContext("2d");
// minimap click-to-pan: convert minimap pixel to world tile, walk there
function mmToWorld(px,py){
  var tx=mmTX0+(px-mmPad)/(mmW-2*mmPad)*(mmTX1-mmTX0);
  var ty=mmTY0+(py-mmPad)/(mmH-2*mmPad)*(mmTY1-mmTY0);
  return {tx:Math.round(tx),ty:Math.round(ty)};
}
var mmDrag=false, mmClickFlash=0, mmClickX=0, mmClickY=0;
mmCv.addEventListener("mousedown",function(e){
  var mmr=mmCv.getBoundingClientRect();
  var mmpx=e.clientX-mmr.left, mmpy=e.clientY-mmr.top;
  var w=mmToWorld(mmpx,mmpy);
  var s=surfaceH(w.tx);
  // SHIFT+click = instant teleport (like RTS minimap)
  if(e.shiftKey){
    player.x=w.tx*TS; player.y=(s-3)*TS; player.vx=0; player.vy=0;
    moveTarget=null;
    showMsg("teleported to ("+w.tx+","+w.ty+")");
  } else {
    moveTarget={tx:w.tx,ty:Math.min(w.ty,s-2)};
  }
  e.stopPropagation();
});
mmCv.addEventListener("mousemove",function(e){
  // Dragging on minimap = walk target follows mouse
  if(!(e.buttons&1)) return;
  var mmr=mmCv.getBoundingClientRect();
  var mmpx=e.clientX-mmr.left, mmpy=e.clientY-mmr.top;
  var w=mmToWorld(mmpx,mmpy);
  var s=surfaceH(w.tx);
  if(e.shiftKey){ player.x=w.tx*TS; player.y=(s-3)*TS; player.vx=0; player.vy=0; }
  else { moveTarget={tx:w.tx,ty:Math.min(w.ty,s-2)}; }
  e.stopPropagation();
});
var mmW=200, mmH=110, mmPad=4;
var mmTX0=-80, mmTX1=120, mmTY0=-24, mmTY1=48;
// agent colour derived from name hash — no static palette; any spawned pilot gets a unique colour
function agentColor(name){
  var h=0; for(var i=0;i<name.length;i++){h=name.charCodeAt(i)+((h<<5)-h);h|=0;}
  // golden hue bias: shift toward amber/orange range, avoid grays
  var hue=((h>>>0)%360+360)%360; if(hue<20||hue>340) hue=30+(Math.abs(h)%40);
  var sat=60+(Math.abs((h>>>8))%35); var lit=55+(Math.abs((h>>>16))%25);
  return "hsl("+hue+","+sat+"%,"+lit+"%)";
}
var agentColorCache={};
function getAgentColor(name){ return agentColorCache[name]||(agentColorCache[name]=agentColor(name)); }
function worldToMM(tx,ty){ return {x:mmPad+(tx-mmTX0)/(mmTX1-mmTX0)*(mmW-2*mmPad), y:mmPad+(ty-mmTY0)/(mmTY1-mmTY0)*(mmH-2*mmPad)}; }
function drawMinimap(t){
  mmCx.clearRect(0,0,mmW,mmH);
  mmCx.fillStyle="#0d0f14"; mmCx.fillRect(0,0,mmW,mmH);
  // render actual tile world — each minimap pixel samples the real castle block at that coordinate
  var spanX=mmTX1-mmTX0, spanY=mmTY1-mmTY0;
  for(var px=0; px<mmW; px++){
    var tx=Math.floor(mmTX0+(px/mmW)*spanX);
    for(var py=0; py<mmH; py++){
      var ty=Math.floor(mmTY0+(py/mmH)*spanY);
      var tile=tileAt(tx,ty);
      if(tile!==AIR){
        mmCx.fillStyle=COL[tile]||"#222";
        mmCx.fillRect(px,py,1,1);
      }
    }
  }
  
  // viewport indicator — shows what part of the world is on screen right now
  var vp0=worldToMM(Math.floor(cam.x/TS), Math.floor(cam.y/TS));
  var vp1=worldToMM(Math.floor((cam.x+W)/TS), Math.floor((cam.y+H)/TS));
  mmCx.fillStyle="rgba(220,220,255,0.08)";
  mmCx.fillRect(vp0.x, vp0.y, vp1.x-vp0.x, vp1.y-vp0.y);
  mmCx.strokeStyle="rgba(220,220,255,0.35)";
  mmCx.lineWidth=1.5;
  mmCx.strokeRect(vp0.x, vp0.y, vp1.x-vp0.x, vp1.y-vp0.y);
  mmCx.lineWidth=1;

  // territory clusters — group rooms by creator, show each pilot's domain
  var territories={};
  rooms.forEach(function(r,i){
    var who=r.n||"unknown";
    if(!territories[who]) territories[who]={rooms:[],cx:0,cy:0};
    territories[who].rooms.push(i);
    territories[who].cx+=r.x;
    territories[who].cy+=r.y;
  });
  for(var who in territories){
    var t=territories[who];
    t.cx/=t.rooms.length;
    t.cy/=t.rooms.length;
    var col=getAgentColor(who);
    var cp=worldToMM(Math.round(t.cx),Math.round(t.cy));
    var radius=5+Math.min(t.rooms.length,30);
    mmCx.fillStyle=col.replace("hsl","hsla").replace(")",",0.12)");
    mmCx.beginPath(); mmCx.arc(cp.x,cp.y,radius,0,6.3); mmCx.fill();
    mmCx.strokeStyle=col.replace("hsl","hsla").replace(")",",0.5)");
    mmCx.lineWidth=1.5; mmCx.stroke(); mmCx.lineWidth=1;
    mmCx.fillStyle=col; mmCx.font="bold 7px monospace"; mmCx.textAlign="center";
    mmCx.fillText(who.slice(0,8),cp.x,cp.y+3);
    mmCx.textAlign="left";
  }

  zones.forEach(function(z){
    var a=worldToMM(z.tx,z.ty), b=worldToMM(z.tx+z.w,z.ty+z.h);
    mmCx.fillStyle="rgba(192,120,40,0.08)"; mmCx.fillRect(a.x,a.y,b.x-a.x,b.y-a.y);
    mmCx.fillStyle="#a88040"; mmCx.font="7px monospace"; mmCx.textAlign="center";
    mmCx.fillText(z.icon, (a.x+b.x)/2, (a.y+b.y)/2+3);
  });
  for(var n in homes){ var h2=homes[n]; if(!h2||h2.tx==null) continue; var hp=worldToMM(h2.tx,h2.ty); mmCx.fillStyle=HOME_COL[n]||"#666"; mmCx.fillRect(hp.x-1.5,hp.y-1.5,3,3); }
  var mmNow=performance.now();
  var mp=worldToMM(Math.floor(player.x/TS),Math.floor(player.y/TS));
  mmCx.fillStyle="#fff"; mmCx.beginPath(); mmCx.arc(mp.x,mp.y,4,0,6.3); mmCx.fill(); mmCx.strokeStyle="#fff"; mmCx.stroke();
  mmCx.fillStyle="#fff"; mmCx.font="8px monospace"; mmCx.textAlign="left"; mmCx.fillText(me.slice(0,8),mp.x+6,mp.y+3);
  for(var w in others){ var o=others[w]; if(mmNow-o.t>8000) continue;
    var op=worldToMM(Math.floor((o.tx||o.x)/TS),Math.floor((o.ty||o.y)/TS));
    var col=getAgentColor(w);
    mmCx.fillStyle=col; mmCx.beginPath(); mmCx.arc(op.x,op.y,3,0,6.3); mmCx.fill();
    mmCx.fillStyle=col; mmCx.font="7px monospace"; mmCx.textAlign="left"; mmCx.fillText(w.slice(0,9),op.x+5,op.y+2);
  }
  mmCx.strokeStyle="#3a2810"; mmCx.lineWidth=2; mmCx.strokeRect(0,0,mmW,mmH);
  mmCx.fillStyle="#a88040"; mmCx.font="bold 8px monospace"; mmCx.textAlign="right"; mmCx.fillText("\u2316 world",mmW-4,10);
  mmCx.textAlign="left";
  // minimap click flash — brief ripple where you last clicked
  if(mmClickFlash && performance.now()-mmClickFlash<600){
    var flashAge=(performance.now()-mmClickFlash)/600;
    mmCx.strokeStyle="rgba(255,255,255,"+(1-flashAge)+")";
    mmCx.lineWidth=2;
    mmCx.beginPath(); mmCx.arc(mmClickX,mmClickY,4+flashAge*20,0,6.3); mmCx.stroke();
    mmCx.strokeStyle="rgba(255,255,255,"+(0.5-flashAge*0.5)+")";
    mmCx.beginPath(); mmCx.arc(mmClickX,mmClickY,2+flashAge*12,0,6.3); mmCx.stroke();
    mmCx.lineWidth=1;
  }
}

// ---- loop removed: heartbeat interval above drives all frames unconditionally ----
// rAF was double-rendering with the heartbeat — second call per frame in focused tabs.
// The heartbeat's setInterval (80ms ≈ 12fps) is the single owner of update/render/hud.
// UNCONDITIONAL heartbeat — starts even if early page code throws.
// rAF throttles to 0fps in unfocused tabs (automation Firefox). This interval
// is the real heartbeat — 12fps always, regardless of visibility. rAF layers on
// top when focused (a human watching). Both call loop(); dt capping prevents
// double-update drift. The castle must render even when no one is looking.
// (heartbeat is now line 1 — no longer needed at end of script)
</script><div id="gm-toggle">☰</div>
<div id="gopher-mirror" style="display:none">
  <div class="header">
    <span class="on" data-col="a">◉ pilot-a</span>
    <span data-col="b">◉ pilot-b</span>
    <span data-col="c">◉ pilot-c</span>
    <span style="margin-left:auto;cursor:pointer;opacity:0.5" id="gm-close">✕</span>
  </div>
  <div class="col on" id="gcol-a"></div>
  <div class="col" id="gcol-b"></div>
  <div class="col" id="gcol-c"></div>
</div>
<script>
// ---- gopher mirror: live stream of all three pilots ----
(function(){
  var mir=document.getElementById("gopher-mirror");
  var tog=document.getElementById("gm-toggle");
  var cols={a:document.getElementById("gcol-a"),b:document.getElementById("gcol-b"),c:document.getElementById("gcol-c")};
  var maxPerCol=150;
  
  tog.addEventListener("click",function(){ mir.style.display=mir.style.display==="none"?"flex":"none"; tog.textContent=mir.style.display==="none"?"◉":"✕"; });
  document.getElementById("gm-close").addEventListener("click",function(){ mir.style.display="none"; tog.textContent="☰"; });
  
  // Tab switching
  mir.querySelectorAll(".header span[data-col]").forEach(function(el){
    el.addEventListener("click",function(){
      mir.querySelectorAll(".header span[data-col]").forEach(function(e){e.classList.remove("on");});
      el.classList.add("on");
      for(var k in cols){ cols[k].classList.toggle("on",k===el.dataset.col); }
    });
  });
  
  // Connect to SSE — LAZILY. Firefox allows only 6 persistent connections per host;
  // with many castle tabs open, an always-on /gophers stream per tab starves the pool
  // and other tabs' /world.json fetches hang forever. Only stream while the mirror is
  // actually visible; close when hidden to return the slot.
  var es=null;
  function gmOpen(){ return mir.style.display!=="none"; }
  function gmSync(){
    if(gmOpen() && !es){ es=connectGophers(); }
    else if(!gmOpen() && es){ es.close(); es=null; }
  }
  tog.addEventListener("click",gmSync);
  document.getElementById("gm-close").addEventListener("click",gmSync);
  setTimeout(gmSync,0); // honor the initial visibility state
  function connectGophers(){
  var es=new EventSource("/gophers");
  es.onmessage=function(m){
    var d; try{d=JSON.parse(m.data);}catch(_){return;}
    var col=cols[d.n.replace("pilot-","")];
    if(!col) return;
    var div=document.createElement("div");
    div.className="gmsg "+d.k;
    div.innerHTML='<span class="gt">'+new Date(d.s).toISOString().slice(11,19)+'</span> '+d.t.slice(0,300);
    col.appendChild(div);
    col.scrollTop=col.scrollHeight;
    // Trim
    while(col.children.length>maxPerCol) col.removeChild(col.firstChild);
  };
  es.onerror=function(){
    // Reconnect handled by EventSource automatically
  };
  return es;
  }
})();
</script>
</body></html>`
