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
  canvas{display:block;width:100vw;height:100vh;image-rendering:pixelated;cursor:crosshair}
  #hud{position:fixed;left:10px;top:8px;color:#d4b464;font:12px 'Courier New',monospace;text-shadow:0 1px 2px #000;pointer-events:none;line-height:1.5}
  #hot{position:fixed;left:50%;bottom:10px;transform:translateX(-50%);display:flex;gap:6px}
  .slot{width:34px;height:34px;border:2px solid #3a2810;background:#181209;display:flex;align-items:center;justify-content:center;font:10px monospace;color:#a88040}
  .slot.on{border-color:#e0a040;box-shadow:0 0 8px #e0a04066}
</style></head>
<body>
<canvas id="c"></canvas>
<div id="hud"></div>
<div id="hot"></div>
<script>
"use strict";
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
var bubbles=[];                     // {who,text,t0}

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
fetch("/world.json").then(function(r){return r.json();}).then(function(d){
  rooms=d.rooms||[]; if(typeof d.seed==="number") seed=d.seed;
  (d.edits||[]).forEach(function(e){ world[e.x+","+e.y]=e.b; });
  buildRooms();
  var s=surfaceH(Math.floor(WW/2)); player.x=(WW/2)*TS; player.y=(s-3)*TS; // spawn center, on ground
});
var es=new EventSource("/events");
es.onmessage=function(m){ var e; try{e=JSON.parse(m.data);}catch(_){return;}
  if(e.type==="edit"){ world[e.x+","+e.y]=e.b; }
  if(e.type==="pos" && e.who!==me){ var o=others[e.who]||(others[e.who]={x:e.x,y:e.y}); o.tx=e.x; o.ty=e.y; o.t=performance.now(); }
  if(e.type==="speech"){ bubbles.push({who:e.who,text:e.text,t0:performance.now()}); }
};
function post(url,obj){ fetch(url,{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify(obj)}).catch(function(){}); }

// ---- player ----
var player={x:WW/2*TS,y:GH*TS,w:20,h:44,vx:0,vy:0,onGround:false};
var keys={};
addEventListener("keydown",function(e){keys[e.key.toLowerCase()]=1; if([" ","arrowup","w","a","d","arrowleft","arrowright"].indexOf(e.key.toLowerCase())>=0)e.preventDefault();});
addEventListener("keyup",function(e){keys[e.key.toLowerCase()]=0;});

var GRAV=1800, MOVE=2600, MAXVX=260, JUMP=560, FRICT=0.80;
function solidAt(px,py){ return !!SOLID[tileAt(Math.floor(px/TS),Math.floor(py/TS))]; }
function hitsWorld(x,y,w,h){ // AABB vs solid tiles
  var x0=Math.floor(x/TS),x1=Math.floor((x+w-1)/TS),y0=Math.floor(y/TS),y1=Math.floor((y+h-1)/TS);
  for(var tx=x0;tx<=x1;tx++)for(var ty=y0;ty<=y1;ty++) if(SOLID[tileAt(tx,ty)]) return true;
  return false;
}
function update(dt){
  var ax=0;
  if(keys["a"]||keys["arrowleft"]) ax-=MOVE;
  if(keys["d"]||keys["arrowright"]) ax+=MOVE;
  player.vx+=ax*dt; if(ax===0) player.vx*=FRICT;
  player.vx=Math.max(-MAXVX,Math.min(MAXVX,player.vx));
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
  if(mouse.btn===0){ // break
    if(SOLID[tileAt(t.tx,t.ty)]){ setTile(t.tx,t.ty,AIR); post("/edit",{who:me,x:t.tx,y:t.ty,b:AIR}); }
  } else if(mouse.btn===2){ // place
    if(tileAt(t.tx,t.ty)===AIR){
      // don't place inside the player
      var pl={x:player.x,y:player.y,w:player.w,h:player.h}, bx=t.tx*TS,by=t.ty*TS;
      if(!(bx<pl.x+pl.w&&bx+TS>pl.x&&by<pl.y+pl.h&&by+TS>pl.y)){ setTile(t.tx,t.ty,hotbar[sel]); post("/edit",{who:me,x:t.tx,y:t.ty,b:hotbar[sel]}); }
    }
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
  cam.x=Math.round(player.x+player.w/2-W/2); cam.y=Math.round(player.y+player.h/2-H/2);
  // sky gradient
  var g=cx.createLinearGradient(0,0,0,H); g.addColorStop(0,"#0b0d12"); g.addColorStop(1,"#1a2230"); cx.fillStyle=g; cx.fillRect(0,0,W,H);
  // visible tiles
  var tx0=Math.floor(cam.x/TS)-1, tx1=Math.floor((cam.x+W)/TS)+1, ty0=Math.floor(cam.y/TS)-1, ty1=Math.floor((cam.y+H)/TS)+1;
  for(var tx=tx0;tx<=tx1;tx++)for(var ty=ty0;ty<=ty1;ty++){ var v=tileAt(tx,ty); if(v!==AIR) drawTile(tx,ty,v); }
  // room labels above their huts
  cx.font="11px Georgia,serif"; cx.textAlign="center";
  rooms.forEach(function(r){ if(r.bx==null)return; var x=(r.bx+2)*TS-cam.x, y=(r.by-1)*TS-cam.y; if(x>-80&&x<W+80){ cx.fillStyle="#a88040"; cx.fillText("R"+rooms.indexOf(r),x,y); } });
  cx.textAlign="left";
  // other players
  var now=performance.now();
  for(var w in others){ var o=others[w]; if(now-o.t>8000){delete others[w];continue;} o.x+=((o.tx||o.x)-o.x)*0.2; o.y+=((o.ty||o.y)-o.y)*0.2; drawGuy(o.x,o.y,"#e0a040",w.slice(0,10)); }
  // me
  drawGuy(player.x,player.y,"#b8ccd8",me.slice(0,10));
  // target highlight
  var tt=targetTile(); if(reachOK(tt.tx,tt.ty)){ cx.strokeStyle=SOLID[tileAt(tt.tx,tt.ty)]?"#e0a040":"#7a9a50"; cx.lineWidth=2; cx.strokeRect(tt.tx*TS-cam.x,tt.ty*TS-cam.y,TS,TS); cx.lineWidth=1; }
  // speech bubbles above whoever spoke
  bubbles=bubbles.filter(function(b){ var age=(now-b.t0)/6000; if(age>1)return false; var who=b.who===me?player:(others[b.who]); if(!who)return true;
    cx.globalAlpha=age<0.85?1:(1-age)/0.15; cx.fillStyle="#1d1509"; cx.strokeStyle="#c07828";
    var tx=(who.x||player.x)-cam.x, ty=(who.y||player.y)-cam.y; var txt=b.text.slice(0,60); var wpx=cx.measureText(txt).width+12;
    cx.fillRect(tx-wpx/2+10,ty-26,wpx,18); cx.strokeRect(tx-wpx/2+10,ty-26,wpx,18); cx.fillStyle="#d4b464"; cx.font="10px Georgia,serif"; cx.textAlign="center"; cx.fillText(txt,tx+10,ty-13); cx.textAlign="left"; cx.globalAlpha=1; return true; });
}

// ---- HUD + hotbar ----
var NAME={3:"stone",5:"plank",4:"wood",2:"dirt"};
function hud(){
  document.getElementById("hud").innerHTML="🏰 pilot castle — the world<br>walk: A/D · jump: W/Space · break: L-click · place: R-click · pick: 1-4<br>you: "+me+"  players: "+(Object.keys(others).length+1)+"  rooms: "+rooms.length;
  var h=document.getElementById("hot"); if(h.childElementCount!==hotbar.length){ h.innerHTML=""; hotbar.forEach(function(b,i){ var d=document.createElement("div"); d.className="slot"; d.textContent=NAME[b]; h.appendChild(d); }); }
  [].forEach.call(h.children,function(el,i){ el.className="slot"+(i===sel?" on":""); });
}

// ---- loop (fixed-ish timestep) ----
var last=performance.now();
function loop(now){
  var dt=Math.min(0.033,(now-last)/1000); last=now;
  if(!document.hidden){ update(dt); syncPos(now); render(now); hud(); }
  requestAnimationFrame(loop);
}
requestAnimationFrame(loop);
</script></body></html>`
