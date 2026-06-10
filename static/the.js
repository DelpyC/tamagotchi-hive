const MAP_W = 60, MAP_H = 40;
let TILE = 18;

// ── Palette ───────────────────────────────────────────────────────
const T_BASE = {
  ocean:     '#163d5c', coast:     '#1e5487',
  grassland: '#2d6b2d', plains:    '#7a8a32',
  desert:    '#c4a44a', forest:    '#1a4d1f',
  hills:     '#6b5c3a', mountain:  '#4e4e5e', tundra: '#7090a0',
};
const T_MID = {
  ocean:     '#1a4a70', coast:     '#2460a0',
  grassland: '#357035', plains:    '#8a9c3a',
  desert:    '#d8b85a', forest:    '#1f5924',
  hills:     '#7a6a46', mountain:  '#5e5e70', tundra: '#80a0b0',
};
const T_HIGH = {
  ocean:     '#204d78', coast:     '#2e70c0',
  grassland: '#3d803d', plains:    '#9aae42',
  desert:    '#e8cc6a', forest:    '#266630',
  hills:     '#8a7a54', mountain:  '#72728a', tundra: '#90b0c4',
};
const T_FR = {
  ocean:'Ocean', coast:'Cote', grassland:'Prairie', plains:'Plaines',
  desert:'Desert', forest:'Foret', hills:'Collines', mountain:'Montagne', tundra:'Toundra',
};
const RES_ICON = {
  wheat:'🌾',cattle:'🐄',fish:'🐟',iron:'⚙',horses:'🐴',
  coal:'◼',gold:'★',silk:'◆',marble:'⬡',
};
const RES_FR = {
  wheat:'Ble',cattle:'Betail',fish:'Poisson',iron:'Fer',
  horses:'Chevaux',coal:'Charbon',gold:'Or',silk:'Soie',marble:'Marbre',
};
const WONDER_ICON = {
  pyramids:'🔺', colosseum:'🏟', oxford_university:'📚',
  eiffel_tower:'🗼', statue_of_liberty:'🗽',
};
const WONDER_IMAGE_SRC = {
  pyramids:'img/buildings/Pyramids.webp',
  colosseum:'img/buildings/Colosseum.webp',
  oxford_university:'img/buildings/Oxford_University.webp',
  eiffel_tower:'img/buildings/Eiffel_Tower.webp',
  statue_of_liberty:'img/buildings/Statue_of_Liberty.webp',
};
const wonderImages = {};
for (const [id, src] of Object.entries(WONDER_IMAGE_SRC)) {
  const im = new Image();
  im.src = src;
  wonderImages[id] = im;
}
const ERA_NAMES  = ['Antiquite','Classique','Medieval','Renaissance'];
const ERA_COLORS = ['#BA7517','#1D9E75','#378ADD','#7F77DD'];
const ERA_BADGE_BG = ['#BA751722','#1D9E7522','#378ADD22','#7F77DD22'];
const BUILDING_COSTS = {
  granary:40,monument:30,library:60,temple:55,barracks:50,
  walls:65,market:60,harbor:70,forge:80,aqueduct:90,settler:70,
  workshop:75,university:120,bank:110,
};
const BUILDING_FR = {
  granary:'Grenier',monument:'Monument',library:'Bibliotheque',
  temple:'Temple',barracks:'Caserne',walls:'Murailles',market:'Marche',
  harbor:'Port',forge:'Forge',aqueduct:'Aqueduc',settler:'Colon',
  workshop:'Atelier',
  university:'Universite',bank:'Banque',
};
const UNIT_ICON = { melee:'⚔', ranged:'🏹', cavalry:'🐎', settler:'👣' };
function uTypeFr(t){
  if (t==='settler') return 'Colon';
  if (t==='ranged') return 'Distance';
  if (t==='cavalry') return 'Cavalerie';
  return 'Melee';
}

// ── Canvas setup ───────────────────────────────────────────────────
const canvas = document.getElementById('map');
const ctx    = canvas.getContext('2d');

// Seeded per-tile noise (deterministic so textures don't flicker)
const tileNoise = [];
for (let i = 0; i < MAP_W * MAP_H * 8; i++) {
  tileNoise.push(Math.sin(i * 127.1 + 311.7) * 0.5 + 0.5);
}
function tn(x, y, k) { return tileNoise[((y * MAP_W + x) * 8 + k) % tileNoise.length]; }

function resize() {
  const sidebar = 260;
  const maxW = Math.min(window.innerWidth - sidebar - 32, 1100);
  TILE = Math.max(14, Math.floor(maxW / MAP_W));
  canvas.width  = TILE * MAP_W;
  canvas.height = TILE * MAP_H;
  needsRender = true;
  if (cur) render(cur);
}
window.addEventListener('resize', resize);

// ── State ──────────────────────────────────────────────────────────
let cur = null;
let fxNow = performance.now();
let lastFrame = 0;
const FX_FRAME_MS_ACTIVE = 120;
const FX_FRAME_MS_IDLE = 450;
const ACTIVE_WINDOW_MS = 2200;
let curCityByPos = {};
let curUnitByPos = {};
let needsRender = true;
let lastStateTs = 0;

// ── WebSocket ──────────────────────────────────────────────────────
const dot = document.getElementById('dot');
const connLbl = document.getElementById('connLbl');
const startScreen = document.getElementById('startScreen');
const startBtn = document.getElementById('startBtn');
const startOptions = [...document.querySelectorAll('.start-option')];
let selectedObjective = 'normal';
let ws;

function showStartScreen() {
  startScreen.classList.remove('hidden');
}

function hideStartScreen() {
  startScreen.classList.add('hidden');
}

function selectObjective(objective) {
  selectedObjective = objective || 'normal';
  for (const item of startOptions) item.classList.toggle('selected', item.dataset.objective === selectedObjective);
}

function objectiveLabel(objective) {
  if (objective === 'domination') return 'Domination';
  if (objective === 'science') return 'Science';
  if (objective === 'culture') return 'Culture';
  if (objective === 'time') return 'Temps';
  return 'Normal';
}

for (const option of startOptions) {
  option.addEventListener('click', () => {
    selectObjective(option.dataset.objective || 'normal');
  });
}

startBtn.addEventListener('click', () => {
  if (ws && ws.readyState === 1) {
    ws.send(`start:${selectedObjective}`);
  }
});

function connect() {
  ws = new WebSocket(`ws://${location.host}/ws`);
  ws.onopen  = () => { dot.className='dot on'; connLbl.textContent='connecte'; };
  ws.onclose = () => { dot.className='dot'; connLbl.textContent='reconnexion...'; setTimeout(connect,2000); };
  ws.onerror = () => ws.close();
  ws.onmessage = evt => {
    cur = JSON.parse(evt.data);
    curCityByPos = {};
    curUnitByPos = {};
    if (cur?.cities) {
      for (const city of cur.cities) curCityByPos[`${city.x},${city.y}`] = city;
    }
    if (cur?.units) {
      for (const u of cur.units) curUnitByPos[`${u.x},${u.y}`] = u;
    }
    lastStateTs = performance.now();
    needsRender = true;
    render(cur);
    updateSidebar(cur);
    updateWonders(cur);
    if (cur.phase === 'setup') showStartScreen();
    if (cur.phase === 'running' || cur.phase === 'ended') hideStartScreen();
    if (document.getElementById('techModal').classList.contains('open')) drawTechTree(cur);
  };
}
document.getElementById('restartBtn').onclick = () => {
  selectObjective('normal');
  showStartScreen();
  if (ws && ws.readyState === 1) ws.send('menu');
};
document.getElementById('techTreeBtn').onclick = () => { document.getElementById('techModal').classList.add('open'); if(cur) drawTechTree(cur); };
document.getElementById('closeTechBtn').onclick = () => { document.getElementById('techModal').classList.remove('open'); };
document.getElementById('techModal').addEventListener('click', e => {
  if (e.target===document.getElementById('techModal')) document.getElementById('techModal').classList.remove('open');
});

// ═══════════════════════════════════════════════════════════════════
//  TERRAIN PAINTERS
// ═══════════════════════════════════════════════════════════════════

function paintTile(x, y, tile) {
  const T   = TILE;
  const px  = x * T, py = y * T;
  const col = T_BASE[tile.terrain] || '#444';
  const mid = T_MID[tile.terrain]  || '#555';
  const hi  = T_HIGH[tile.terrain] || '#666';

  // 1. Base fill
  ctx.fillStyle = col;
  ctx.fillRect(px, py, T, T);

  // 2. Terrain-specific texture
  ctx.save();
  ctx.beginPath();
  ctx.rect(px, py, T, T);
  ctx.clip();

  switch (tile.terrain) {

    case 'ocean': case 'coast': {
      // Layered wave bands
      const wc = tile.terrain === 'ocean' ? mid : hi;
      for (let i = 0; i < 3; i++) {
        const oy = py + T * (0.2 + i * 0.28) + tn(x,y,i) * T * 0.08;
        ctx.beginPath();
        ctx.moveTo(px, oy);
        ctx.bezierCurveTo(px+T*0.25, oy - T*0.06, px+T*0.75, oy + T*0.06, px+T, oy);
        ctx.strokeStyle = wc + (i===1?'66':'44');
        ctx.lineWidth = 1.2;
        ctx.stroke();
      }
      // Slight shimmer dots
      ctx.fillStyle = hi + '33';
      for (let i = 0; i < 4; i++) {
        ctx.beginPath();
        ctx.arc(px + tn(x,y,i)*T, py + tn(x,y,i+4)*T, 1, 0, Math.PI*2);
        ctx.fill();
      }
      break;
    }

    case 'grassland': {
      // Grass tufts - small V strokes
      ctx.strokeStyle = hi + '80';
      ctx.lineWidth = 0.9;
      const pts = [[.15,.7],[.4,.45],[.65,.7],[.85,.55],[.3,.8],[.75,.3]];
      for (const [fx,fy] of pts) {
        const bx=px+fx*T, by=py+fy*T, h=T*0.15;
        ctx.beginPath(); ctx.moveTo(bx-T*0.04,by); ctx.lineTo(bx,by-h); ctx.lineTo(bx+T*0.04,by); ctx.stroke();
      }
      // Subtle diagonal light
      const gl = ctx.createLinearGradient(px,py,px+T,py+T);
      gl.addColorStop(0, hi+'22'); gl.addColorStop(1,'transparent');
      ctx.fillStyle=gl; ctx.fillRect(px,py,T,T);
      break;
    }

    case 'plains': {
      // Horizontal field rows
      for (let i=1;i<5;i++) {
        const ly = py + T*i/5;
        ctx.beginPath(); ctx.moveTo(px,ly); ctx.lineTo(px+T,ly);
        ctx.strokeStyle = (i%2===0?hi:mid)+'44';
        ctx.lineWidth=0.7; ctx.stroke();
      }
      // Diagonal light
      const gp = ctx.createLinearGradient(px,py,px+T,py+T);
      gp.addColorStop(0, hi+'30'); gp.addColorStop(1,'transparent');
      ctx.fillStyle=gp; ctx.fillRect(px,py,T,T);
      break;
    }

    case 'desert': {
      // Sand dune arcs
      for (let i=0;i<2;i++) {
        const dy2 = py + T*(0.35+i*0.38);
        ctx.beginPath();
        ctx.moveTo(px, dy2 + tn(x,y,i)*T*0.1);
        ctx.quadraticCurveTo(px+T*0.5, dy2-T*0.18, px+T, dy2+tn(x,y,i+2)*T*0.1);
        ctx.strokeStyle = hi+'66'; ctx.lineWidth=1.2; ctx.stroke();
      }
      // Hot shimmer gradient
      const gd = ctx.createLinearGradient(px,py,px,py+T);
      gd.addColorStop(0,hi+'44'); gd.addColorStop(0.4,mid+'00'); gd.addColorStop(1,col+'00');
      ctx.fillStyle=gd; ctx.fillRect(px,py,T,T);
      break;
    }

    case 'forest': {
      // Pine tree cluster
      ctx.fillStyle = hi + 'aa';
      const trees = [[.22,.78],[.5,.55],[.78,.78],[.35,.6],[.65,.6]];
      const ts = T * 0.2;
      for (const [fx,fy] of trees) {
        const tx=px+fx*T, ty=py+fy*T;
        ctx.beginPath();
        ctx.moveTo(tx, ty-ts*1.2);
        ctx.lineTo(tx-ts*0.7, ty+ts*0.4);
        ctx.lineTo(tx+ts*0.7, ty+ts*0.4);
        ctx.closePath(); ctx.fill();
        // Trunk
        ctx.fillStyle = '#2a1a0a88';
        ctx.fillRect(tx-ts*0.1, ty+ts*0.4, ts*0.2, ts*0.4);
        ctx.fillStyle = hi + 'aa';
      }
      // Dark canopy overlay
      const gf = ctx.createRadialGradient(px+T/2,py+T/2,0,px+T/2,py+T/2,T*0.7);
      gf.addColorStop(0,'transparent'); gf.addColorStop(1,'#000000' + '33');
      ctx.fillStyle=gf; ctx.fillRect(px,py,T,T);
      break;
    }

    case 'hills': {
      // Two overlapping hill humps
      const drawHump = (cx2,cy2,rx,ry,alpha) => {
        const g = ctx.createRadialGradient(cx2,cy2-ry*0.3,ry*0.1,cx2,cy2,rx);
        g.addColorStop(0, hi+alpha); g.addColorStop(1,col+'00');
        ctx.fillStyle = g;
        ctx.beginPath(); ctx.ellipse(cx2,cy2,rx,ry,0,0,Math.PI*2); ctx.fill();
      };
      drawHump(px+T*0.32, py+T*0.72, T*0.3, T*0.22,'cc');
      drawHump(px+T*0.70, py+T*0.65, T*0.26, T*0.19,'aa');
      // Shadow at base
      const gs = ctx.createLinearGradient(px,py+T*0.7,px,py+T);
      gs.addColorStop(0,'transparent'); gs.addColorStop(1,'#00000033');
      ctx.fillStyle=gs; ctx.fillRect(px,py,T,T);
      break;
    }

    case 'mountain': {
      // Main peak with snow
      const mx=px+T*0.5, peakY=py+T*0.1, baseY=py+T*0.9;
      // Shadow side (left)
      ctx.beginPath(); ctx.moveTo(mx,peakY); ctx.lineTo(px+T*0.08,baseY); ctx.lineTo(mx,baseY); ctx.closePath();
      ctx.fillStyle='#00000055'; ctx.fill();
      // Light side (right)  
      ctx.beginPath(); ctx.moveTo(mx,peakY); ctx.lineTo(mx,baseY); ctx.lineTo(px+T*0.92,baseY); ctx.closePath();
      ctx.fillStyle=mid+'bb'; ctx.fill();
      // Outline
      ctx.beginPath(); ctx.moveTo(mx,peakY); ctx.lineTo(px+T*0.08,baseY); ctx.lineTo(px+T*0.92,baseY); ctx.closePath();
      ctx.strokeStyle='#00000066'; ctx.lineWidth=0.8; ctx.stroke();
      // Snow cap
      const snowLine = py+T*0.36;
      ctx.beginPath(); ctx.moveTo(mx,peakY);
      ctx.lineTo(px+T*0.36,snowLine); ctx.lineTo(px+T*0.64,snowLine); ctx.closePath();
      ctx.fillStyle='#ffffffcc'; ctx.fill();
      break;
    }

    case 'tundra': {
      // Cracked ice pattern
      ctx.strokeStyle = hi + '55';
      ctx.lineWidth = 0.7;
      const cracks = [
        [[.1,.3],[.4,.55],[.7,.4]],
        [[.2,.7],[.5,.55],[.8,.75]],
        [[.55,.2],[.6,.55]],
      ];
      for (const pts of cracks) {
        ctx.beginPath();
        ctx.moveTo(px+pts[0][0]*T, py+pts[0][1]*T);
        for (let i=1;i<pts.length;i++) ctx.lineTo(px+pts[i][0]*T, py+pts[i][1]*T);
        ctx.stroke();
      }
      // Frost overlay
      const gt = ctx.createLinearGradient(px,py,px+T,py+T);
      gt.addColorStop(0,hi+'33'); gt.addColorStop(1,'transparent');
      ctx.fillStyle=gt; ctx.fillRect(px,py,T,T);
      break;
    }
  }

  // 3. Isometric edge shading - makes the map feel slightly 3D
  // Right and bottom edges are darker (shadow), top and left lighter
  const edgeW = Math.max(2, T * 0.08);
  // Top-left highlight
  ctx.fillStyle = '#ffffff12';
  ctx.fillRect(px, py, T, edgeW);
  ctx.fillRect(px, py, edgeW, T);
  // Bottom-right shadow
  ctx.fillStyle = '#00000028';
  ctx.fillRect(px, py+T-edgeW, T, edgeW);
  ctx.fillRect(px+T-edgeW, py, edgeW, T);

  ctx.restore();
}

// ═══════════════════════════════════════════════════════════════════
//  CITY PAINTER - Civ6-inspired district silhouette
// ═══════════════════════════════════════════════════════════════════

function paintCity(city, civColor) {
  const T   = TILE;
  const px  = city.x * T, py = city.y * T;
  const cx  = px + T/2, cy = py + T/2;
  const col = civColor[city.civId] || '#aaa';
  const pop = city.population;

  const hasWalls   = city.buildings?.includes('walls');
  const hasHarbor  = city.buildings?.includes('harbor') && city.isCoastal;
  const hasLibrary = city.buildings?.includes('library') || city.buildings?.includes('university');
  const hasTemple  = city.buildings?.includes('temple');
  const hasMarket  = city.buildings?.includes('market') || city.buildings?.includes('bank');
  const hasForge   = city.buildings?.includes('forge') || city.buildings?.includes('workshop');

  // Scale with population
  const baseS = T * Math.min(0.40, 0.20 + pop * 0.025);

  ctx.save();

  // ── Ground shadow ──
  ctx.shadowColor = '#000000aa';
  ctx.shadowBlur  = T * 0.5;
  ctx.shadowOffsetX = T * 0.06;
  ctx.shadowOffsetY = T * 0.06;

  // ── Walls ring (if built) ──
  if (hasWalls) {
    const wr = baseS + T * 0.12;
    // Outer wall
    ctx.strokeStyle = '#c8b890';
    ctx.lineWidth = T * 0.06;
    ctx.strokeRect(cx-wr, cy-wr, wr*2, wr*2);
    // Inner wall face
    ctx.strokeStyle = '#a09070';
    ctx.lineWidth = T * 0.03;
    ctx.strokeRect(cx-wr+T*0.025, cy-wr+T*0.025, (wr-T*0.025)*2, (wr-T*0.025)*2);
    // Battlement notches along top
    ctx.fillStyle = '#c8b890';
    const nCount = 6, nW = T*0.03, nH = T*0.04;
    for (let i=0;i<=nCount;i++) {
      const nx2 = cx - wr + (wr*2/nCount)*i;
      ctx.fillRect(nx2-nW/2, cy-wr-nH, nW, nH);
      ctx.fillRect(cx-wr-nH, cy-wr+(wr*2/nCount)*i-nW/2, nH, nW);
    }
  }

  ctx.shadowBlur = T * 0.3;

  // ── City center keep ──
  const ks = baseS;
  // Stone base
  const grad = ctx.createLinearGradient(cx-ks, cy-ks, cx+ks, cy+ks);
  grad.addColorStop(0, '#2a2218');
  grad.addColorStop(0.3, col + 'dd');
  grad.addColorStop(1, col + '88');
  ctx.fillStyle = grad;
  ctx.fillRect(cx-ks, cy-ks, ks*2, ks*2);
  // Keep edge
  ctx.strokeStyle = '#ffffff55';
  ctx.lineWidth = 1;
  ctx.strokeRect(cx-ks, cy-ks, ks*2, ks*2);

  // ── Gate ──
  const gw = ks*0.38, gh = ks*0.52;
  ctx.fillStyle = '#00000099';
  ctx.fillRect(cx-gw/2, cy+ks-gh, gw, gh);
  ctx.beginPath(); ctx.arc(cx, cy+ks-gh, gw/2, Math.PI, 0);
  ctx.fillStyle='#00000099'; ctx.fill();

  ctx.shadowBlur = 0;

  // ── Corner towers (pop ≥ 2) ──
  if (pop >= 2) {
    const ts = ks * 0.30;
    const corners = [
      [cx-ks-ts*0.5, cy-ks-ts*0.5],
      [cx+ks-ts*0.5, cy-ks-ts*0.5],
      [cx-ks-ts*0.5, cy+ks-ts*0.5],
      [cx+ks-ts*0.5, cy+ks-ts*0.5],
    ];
    for (const [tx,ty] of corners) {
      ctx.fillStyle='#1a130e';
      ctx.fillRect(tx, ty, ts, ts);
      ctx.fillStyle=col+'cc';
      ctx.fillRect(tx+1, ty+1, ts-2, ts-2);
      // Merlons
      ctx.fillStyle='#0009';
      ctx.fillRect(tx+1, ty-ts*0.25, ts*0.3, ts*0.28);
      ctx.fillRect(tx+ts*0.7, ty-ts*0.25, ts*0.3, ts*0.28);
    }
  }

  // District icons - small symbols beside the keep
  // Each district appears as a tiny colored square with icon
  const districts = [];
  if (hasLibrary)  districts.push(['📚','#378ADD']);
  if (hasTemple)   districts.push(['⛪','#BA7517']);
  if (hasMarket)   districts.push(['💰','#1D9E75']);
  if (hasForge)    districts.push(['⚒','#E24B4A']);
  if (hasHarbor)   districts.push(['H','#2255a0']);

  if (districts.length > 0) {
    const ds = T * 0.22;
    let dOffset = -(districts.length * (ds+2)) / 2;
    for (const [icon, dcol] of districts) {
      const dx = cx + dOffset + ds/2;
      const dy = cy - ks - ds - T*0.08;
      ctx.fillStyle = '#0009';
      ctx.fillRect(dx-ds/2+1, dy-ds/2+1, ds, ds);
      ctx.fillStyle = dcol + 'cc';
      ctx.fillRect(dx-ds/2, dy-ds/2, ds, ds);
      ctx.strokeStyle='#ffffff44'; ctx.lineWidth=0.5;
      ctx.strokeRect(dx-ds/2, dy-ds/2, ds, ds);
      ctx.font = `${Math.floor(ds*0.7)}px serif`;
      ctx.textAlign='center'; ctx.textBaseline='middle';
      ctx.fillText(icon, dx, dy);
      dOffset += ds + 2;
    }
  }

  // ── Harbor mast ──
  if (hasHarbor) {
    const hx = cx + ks + T*0.12, hy = cy - ks*0.6;
    ctx.strokeStyle='#c8a050'; ctx.lineWidth=1.2;
    ctx.beginPath(); ctx.moveTo(hx, cy+ks*0.2); ctx.lineTo(hx, hy); ctx.stroke();
    ctx.beginPath(); ctx.moveTo(hx, hy); ctx.lineTo(hx+T*0.22, hy+T*0.14); ctx.stroke();
    ctx.fillStyle=col+'cc';
    ctx.beginPath(); ctx.moveTo(hx, hy); ctx.lineTo(hx+T*0.2, hy+T*0.1); ctx.lineTo(hx, hy+T*0.22); ctx.closePath(); ctx.fill();
  }

  // ── City name banner ──
  const labelY = cy + ks + (hasWalls ? T*0.2 : T*0.1) + 4;
  const cityFont = `${Math.max(8, Math.floor(T*0.32))}px 'Cinzel',serif`;
  ctx.font = cityFont;
  ctx.textAlign = 'center';
  ctx.textBaseline = 'top';
  const nameW = ctx.measureText(city.name).width;
  const bannerPad = 4;
  const bannerH   = Math.max(8, Math.floor(T*0.32)) + bannerPad*2;
  // Banner background
  ctx.fillStyle = '#0009';
  ctx.fillRect(cx - nameW/2 - bannerPad - 1, labelY - 1, nameW + bannerPad*2 + 2, bannerH + 2);
  ctx.fillStyle = col + '99';
  ctx.fillRect(cx - nameW/2 - bannerPad, labelY, nameW + bannerPad*2, bannerH);
  // Left/right color bars
  ctx.fillStyle = col;
  ctx.fillRect(cx - nameW/2 - bannerPad, labelY, 3, bannerH);
  ctx.fillRect(cx + nameW/2 + bannerPad - 3, labelY, 3, bannerH);
  // Name text
  ctx.fillStyle = '#fff';
  ctx.font = cityFont;
  ctx.fillText(city.name, cx, labelY + bannerPad);

  ctx.restore();
}

// ═══════════════════════════════════════════════════════════════════
//  MAIN RENDER
// ═══════════════════════════════════════════════════════════════════

function render(s) {
  if (!TILE || TILE < 1) return;
  const T = TILE;
  const pulse = 0.5 + 0.5 * Math.sin(fxNow / 560);
  const roadScroll = -(fxNow / 85) % (T * 0.43);
  ctx.clearRect(0, 0, canvas.width, canvas.height);

  const civColor = {};
  for (const c of s.civs) civColor[c.id] = c.color;
  // ── PASS 1: terrain ──────────────────────────────────────────────
  for (let y = 0; y < MAP_H; y++) {
    for (let x = 0; x < MAP_W; x++) {
      paintTile(x, y, s.tiles[y][x]);
    }
  }

  // ── PASS 2: territory soft glow ───────────────────────────────────
  // Draw each claimed tile with a blurred civ-color overlay
  for (let y = 0; y < MAP_H; y++) {
    for (let x = 0; x < MAP_W; x++) {
      const tile = s.tiles[y][x];
      if (tile.civId === -1) continue;
      const col = civColor[tile.civId] || '#fff';
      const alpha = 0.12 + pulse * 0.08;
      ctx.fillStyle = `${col}${Math.floor(alpha * 255).toString(16).padStart(2,'0')}`;
      ctx.fillRect(x*T, y*T, T, T);
    }
  }

  // ── PASS 3: territory borders (painted edge style) ────────────────
  for (let y = 0; y < MAP_H; y++) {
    for (let x = 0; x < MAP_W; x++) {
      const civ = s.tiles[y][x].civId;
      if (civ === -1) continue;
      const col = civColor[civ] || '#fff';
      const px = x*T, py = y*T;

      const right  = x+1<MAP_W && s.tiles[y][x+1].civId !== civ;
      const bottom = y+1<MAP_H && s.tiles[y+1][x].civId !== civ;
      const left   = x===0     || s.tiles[y][x-1].civId !== civ;
      const top    = y===0     || s.tiles[y-1][x].civId !== civ;

      const paintEdge = (x1,y1,x2,y2) => {
        ctx.strokeStyle = col + 'ee';
        ctx.lineWidth = 3.5;
        ctx.lineCap = 'round';
        ctx.beginPath(); ctx.moveTo(x1,y1); ctx.lineTo(x2,y2); ctx.stroke();
        ctx.strokeStyle = '#fff5';
        ctx.lineWidth = 1;
        ctx.beginPath(); ctx.moveTo(x1,y1); ctx.lineTo(x2,y2); ctx.stroke();
      };
      if (right)  paintEdge(px+T, py,     px+T, py+T);
      if (bottom) paintEdge(px,   py+T,   px+T, py+T);
      if (left)   paintEdge(px,   py,     px,   py+T);
      if (top)    paintEdge(px,   py,     px+T, py);
    }
  }

  // ── PASS 4: roads (dashed, between same-civ cities) ───────────────
  const drawn = new Set();
  for (let i = 0; i < s.cities.length; i++) {
    const a = s.cities[i];
    for (let j = i+1; j < s.cities.length; j++) {
      const b = s.cities[j];
      if (a.civId !== b.civId) continue;
      const key = `${a.id}-${b.id}`;
      if (drawn.has(key)) continue;
      drawn.add(key);
      const ax=a.x*T+T/2, ay=a.y*T+T/2;
      const bx=b.x*T+T/2, by=b.y*T+T/2;
      if (Math.hypot(ax-bx,ay-by) > T*15) continue;
      ctx.setLineDash([T*0.25, T*0.18]);
      ctx.lineDashOffset = roadScroll;
      ctx.strokeStyle = '#c8a05077';
      ctx.lineWidth = 1.8;
      ctx.lineCap = 'round';
      ctx.beginPath(); ctx.moveTo(ax,ay); ctx.lineTo(bx,by); ctx.stroke();
      ctx.setLineDash([]);
      ctx.lineDashOffset = 0;
    }
  }

  // ── PASS 5: resources ──────────────────────────────────────────────
  for (let y = 0; y < MAP_H; y++) {
    for (let x = 0; x < MAP_W; x++) {
      const tile = s.tiles[y][x];
      if (!tile.resource || !tile.visible) continue;
      const icon = RES_ICON[tile.resource];
      if (!icon) continue;
      const px=x*T, py=y*T;
      const bx=px+T*0.78, by=py+T*0.22, br=T*0.17;
      const bob = Math.sin((fxNow / 310) + x * 0.35 + y * 0.22) * T * 0.03;
      // Badge
      ctx.fillStyle='#000c';
      ctx.beginPath(); ctx.arc(bx,by+bob,br+1.5,0,Math.PI*2); ctx.fill();
      ctx.fillStyle='#ffffffcc';
      ctx.beginPath(); ctx.arc(bx,by+bob,br,0,Math.PI*2); ctx.fill();
      ctx.font=`${Math.floor(T*0.26)}px serif`;
      ctx.textAlign='center'; ctx.textBaseline='middle';
      ctx.fillText(icon, bx, by+bob);
    }
  }

  // ── PASS 6: cities ─────────────────────────────────────────────────
  // Sort by Y so northern cities render under southern ones
  const sortedCities = [...s.cities].sort((a,b) => a.y - b.y);
  for (const city of sortedCities) {
    paintCity(city, civColor);
  }

  // ── PASS 7: units (so spectators can follow wars) ───────────────────
  if (s.units) {
    for (const u of s.units) {
      const px=u.x*T, py=u.y*T;
      const cx=px+T*0.5, cy=py+T*0.46;
      const col = civColor[u.civId] || '#fff';
      ctx.fillStyle = '#000a';
      ctx.beginPath(); ctx.arc(cx+1,cy+2,T*0.2,0,Math.PI*2); ctx.fill();
      ctx.fillStyle = col;
      ctx.beginPath(); ctx.arc(cx,cy,T*0.2,0,Math.PI*2); ctx.fill();
      ctx.strokeStyle = '#fff8';
      ctx.lineWidth = 1;
      ctx.stroke();

      // Unit class icon.
      ctx.font=`${Math.floor(T*0.18)}px serif`;
      ctx.textAlign='center'; ctx.textBaseline='middle';
      ctx.fillStyle='#f5f1d5';
      ctx.fillText(UNIT_ICON[u.type]||'⚔', cx, cy);

      if (u.type !== 'settler') {
        // Strength pips
        const pips = Math.max(1, Math.min(5, Math.floor((u.strength||1)+0.2)));
        for (let i=0;i<pips;i++) {
          ctx.fillStyle = '#fff';
          ctx.fillRect(px+T*0.18+i*T*0.08, py+T*0.72, T*0.05, T*0.05);
        }
      }
    }
  }

  // ── PASS 8: world wonders on their built city tiles ────────────────
  if (s.wonders) {
    const cityById = {};
    for (const city of s.cities) cityById[city.id] = city;
    for (const w of s.wonders) {
      if (w.civId === -1) continue;
      const city = cityById[w.cityId];
      if (!city) continue;
      const px=city.x*T, py=city.y*T;
      const img = wonderImages[w.id];
      if (img && img.complete && img.naturalWidth > 0) {
        const iw = T*2, ih = T*2;
        ctx.shadowColor='#0008';
        ctx.shadowBlur=5;
        ctx.drawImage(img, px + (T - iw)/2, py + (T - ih)/2, iw, ih);
        ctx.shadowBlur=0;
      } else {
        ctx.font=`${Math.floor(T*0.3)}px serif`;
        ctx.textAlign='center'; ctx.textBaseline='middle';
        const sparkle = 0.15 + pulse * 0.2;
        ctx.shadowColor='#f0c04099'; ctx.shadowBlur=8 + pulse*6;
        ctx.globalAlpha = 0.8 + sparkle;
        ctx.fillText(WONDER_ICON[w.id]||'✨', px+T*0.18, py+T*0.18);
        ctx.globalAlpha = 1;
        ctx.shadowBlur=0;
      }
    }
  }

  // ── PASS 9: fog of war ─────────────────────────────────────────────
  for (let y = 0; y < MAP_H; y++) {
    for (let x = 0; x < MAP_W; x++) {
      const tile = s.tiles[y][x];
      const px=x*T, py=y*T;
      if (!tile.explored) {
        // Completely dark - not yet discovered
        ctx.fillStyle = '#00000099';
        ctx.fillRect(px, py, T, T);
        // Slight noise texture to avoid flat black
        ctx.fillStyle = `rgba(0,0,0,${0.82 + tn(x,y,0)*0.12})`;
        ctx.fillRect(px, py, T, T);
      } else if (!tile.visible) {
        // Explored but out of sight - desaturated dark overlay
        ctx.fillStyle = '#000000' + '60';
        ctx.fillRect(px, py, T, T);
        // Bluish tint to feel "cold/past"
        ctx.fillStyle = '#0a1a3a22';
        ctx.fillRect(px, py, T, T);
      }
    }
  }

  // ── PASS 10: soft vignette around map edges ────────────────────────
  const vW = canvas.width, vH = canvas.height;
  const vig = ctx.createRadialGradient(vW/2,vH/2,vH*0.35,vW/2,vH/2,vH*0.78);
  vig.addColorStop(0,'transparent');
  vig.addColorStop(1,'#00000055');
  ctx.fillStyle=vig; ctx.fillRect(0,0,vW,vH);

  // Moving atmospheric shadows for more cinematic motion while spectating.
  const cloudShift = (fxNow / 28) % (T * 18);
  const cloud = ctx.createLinearGradient(-cloudShift, 0, vW - cloudShift, vH);
  cloud.addColorStop(0, 'rgba(255,255,255,0.00)');
  cloud.addColorStop(0.25, 'rgba(255,255,255,0.03)');
  cloud.addColorStop(0.5, 'rgba(0,0,0,0.05)');
  cloud.addColorStop(0.75, 'rgba(255,255,255,0.02)');
  cloud.addColorStop(1, 'rgba(0,0,0,0.06)');
  ctx.fillStyle = cloud;
  ctx.fillRect(0, 0, vW, vH);
  needsRender = false;
}

// ═══════════════════════════════════════════════════════════════════
//  SIDEBAR
// ═══════════════════════════════════════════════════════════════════

function updateSidebar(s) {
  document.getElementById('turnNum').textContent = s.turn;
  const maxEra = Math.max(...s.civs.map(c => c.era||0));
  const eraLabel = document.getElementById('eraLabel');
  if (s.phase === 'ended') {
    eraLabel.textContent = `FIN - ${s.victory||''}`;
    eraLabel.style.color = '#f0c060';
  } else {
    eraLabel.textContent = ERA_NAMES[maxEra]||'';
    eraLabel.style.color = '';
  }

  const civList = document.getElementById('civList');
  civList.innerHTML = '';
  const ranking = [];

  for (const civ of s.civs) {
    let pop=0;
    for (const city of s.cities) if(city.civId===civ.id) pop+=city.population;
    const power = civ.science + civ.gold + pop*12 + civ.cities.length*25 + (civ.era||0)*50;
    ranking.push({ civ, pop, power });

    let resHTML='';
    if (civ.currentResearch && s.techTree) {
      const td=s.techTree.find(t=>t.id===civ.currentResearch);
      if (td) {
        const pct=Math.min(100,Math.max(0,civ.researchProg||0));
        resHTML=`<div class="civ-research">Recherche: ${td.name} (${pct}%)</div>
          <div class="research-bar-wrap"><div class="research-bar-fill" style="width:${pct}%"></div></div>`;
      }
    }
    const eraBg=ERA_BADGE_BG[civ.era||0]||'#21293a';
    const eraCol=ERA_COLORS[civ.era||0]||'#7a8fa8';
    const eraName=ERA_NAMES[civ.era||0]||'';
    const row=document.createElement('div');
    row.className='civ-row';
    row.innerHTML=`
      <div class="civ-dot" style="background:${civ.color}"></div>
      <div style="flex:1;min-width:0;">
        <div style="display:flex;align-items:center;gap:6px;">
          <span class="civ-name">${civ.name}</span>
          <span class="civ-era-badge" style="background:${eraBg};color:${eraCol};border:1px solid ${eraCol}44">${eraName}</span>
        </div>
        <div class="civ-stats">Villes ${civ.cities.length} | Pop ${pop} | Or ${civ.gold} | Science ${civ.science}</div>
        ${resHTML}
      </div>`;
    civList.appendChild(row);
  }

  ranking.sort((a,b)=>b.power-a.power);
  const powerBoard = document.getElementById('powerBoard');
  if (powerBoard) {
    powerBoard.innerHTML = '';
    for (let i = 0; i < Math.min(5, ranking.length); i++) {
      const r = ranking[i];
      const pRow = document.createElement('div');
      pRow.className = 'power-row';
      pRow.innerHTML = `
        <div class="power-rank">#${i+1}</div>
        <div class="power-name"><span class="power-dot" style="background:${r.civ.color}"></span>${r.civ.name}</div>
        <div class="power-score">${r.power}</div>`;
      powerBoard.appendChild(pRow);
    }
  }

  const victoryBoard = document.getElementById('victoryBoard');
  if (victoryBoard) {
    const wonderByCiv = {};
    for (const c of s.civs) wonderByCiv[c.id] = 0;
    for (const w of (s.wonders||[])) if (w.civId !== -1) wonderByCiv[w.civId] = (wonderByCiv[w.civId]||0)+1;

    const scienceTarget = 18;
    const cultureTarget = 3;
    const topScience = Math.max(...s.civs.map(c => (c.knownTechs||[]).length));
    const topCulture = Math.max(...s.civs.map(c => wonderByCiv[c.id]||0));
    const aliveCount = s.civs.filter(c=>c.alive).length;
    const cityCounts = {};
    for (const c of s.civs) cityCounts[c.id] = 0;
    for (const city of s.cities) cityCounts[city.civId] = (cityCounts[city.civId]||0)+1;
    const topCities = Math.max(0, ...Object.values(cityCounts));

    const pScience = Math.min(100, Math.round((topScience/scienceTarget)*100));
    const pCulture = Math.min(100, Math.round((topCulture/cultureTarget)*100));
    const eliminationDom = (s.civs.length-aliveCount)/(s.civs.length-1);
    const cityDom = s.cities.length ? topCities/s.cities.length : 0;
    const pDom = s.phase === 'ended' && s.victory === 'Domination'
      ? 100
      : Math.min(100, Math.round(Math.max(eliminationDom, cityDom)*100));
    const pTime = Math.min(100, Math.round((s.turn/280)*100));

    victoryBoard.innerHTML = `
      <div class="objective-line">Objectif : ${objectiveLabel(s.objective)}</div>
      <div class="v-row"><span>Domination</span><div class="v-bar"><div style="width:${pDom}%"></div></div><b>${pDom}%</b></div>
      <div class="v-row"><span>Science</span><div class="v-bar"><div style="width:${pScience}%"></div></div><b>${topScience}/${scienceTarget}</b></div>
      <div class="v-row"><span>Culture</span><div class="v-bar"><div style="width:${pCulture}%"></div></div><b>${topCulture}/${cultureTarget}</b></div>
      <div class="v-row"><span>Temps</span><div class="v-bar"><div style="width:${pTime}%"></div></div><b>${s.turn}/280</b></div>`;
  }

  const log=document.getElementById('eventLog');
  log.innerHTML='';
  const rev=[...s.events].reverse();
  for (let i=0;i<rev.length;i++) {
    const div=document.createElement('div');
    div.className='event'+(i===0?' new':'')+(rev[i].includes('✨')?' wonder':'');
    div.textContent=rev[i];
    log.appendChild(div);
  }

  const leg=document.getElementById('legend');
  if (!leg.childElementCount) {
    for (const [k,col] of Object.entries(T_BASE)) {
      leg.innerHTML+=`<div class="leg"><div class="leg-sq" style="background:${col}"></div>${T_FR[k]||k}</div>`;
    }
  }
}

// ── Wonders ────────────────────────────────────────────────────────
function updateWonders(s) {
  const list=document.getElementById('wonderList');
  if(!list||!s.wonders) return;
  const civColor={},civName={};
  for(const c of s.civs){civColor[c.id]=c.color;civName[c.id]=c.name;}
  list.innerHTML='';
  for(const w of s.wonders){
    const icon=WONDER_ICON[w.id]||'🏛';
    const imgSrc=WONDER_IMAGE_SRC[w.id]||'';
    const iconHTML=imgSrc
      ? `<img class="wonder-icon-img" src="${imgSrc}" alt="${w.name}">`
      : icon;
    const row=document.createElement('div');
    row.className='wonder-row';
    if(w.civId!==-1){
      const col=civColor[w.civId]||'#fff',name=civName[w.civId]||'?';
      row.innerHTML=`<div class="wonder-icon">${iconHTML}</div>
        <div style="flex:1;min-width:0;">
          <div class="wonder-name">${w.name}</div>
          <div class="wonder-owner" style="color:${col}">${name} - Tour ${w.turn}</div>
        </div>`;
    } else {
      row.innerHTML=`<div class="wonder-icon" style="opacity:.4">${iconHTML}</div>
        <div style="flex:1;min-width:0;">
          <div class="wonder-name unbuilt">${w.name}</div>
          <div class="wonder-owner" style="color:#7a8fa8">Non construite</div>
        </div>`;
    }
    list.appendChild(row);
  }
}

// ── Tooltip ────────────────────────────────────────────────────────
const tooltip=document.getElementById('tooltip');
canvas.addEventListener('mousemove', e => {
  if(!cur) return;
  const rect=canvas.getBoundingClientRect();
  const tx=Math.floor((e.clientX-rect.left)*(canvas.width/rect.width)/TILE);
  const ty=Math.floor((e.clientY-rect.top)*(canvas.height/rect.height)/TILE);
  if(tx<0||tx>=MAP_W||ty<0||ty>=MAP_H){tooltip.style.display='none';return;}
  const tile=cur.tiles[ty][tx];
  const city=curCityByPos[`${tx},${ty}`];
  const unit=curUnitByPos[`${tx},${ty}`]||null;

  if(!tile.explored){
    tooltip.innerHTML='<div class="tt-title">Territoire Inexplore</div>';
    tooltip.style.display='block';
  } else {
    let html=`<div class="tt-title">${T_FR[tile.terrain]||tile.terrain}</div>`;
    if(tile.resource) html+=`<div>Ressource : ${RES_FR[tile.resource]||tile.resource} ${RES_ICON[tile.resource]||''}</div>`;
    if(tile.civId!==-1){const civ=cur.civs[tile.civId];if(civ) html+=`<div class="tt-muted">Territoire : <span style="color:${civ.color}">${civ.name}</span></div>`;}
    if(city){
      const civ=cur.civs[city.civId];
      html+=`<hr class="tt-divider">`;
      html+=`<div class="tt-title" style="color:${civ?.color||'#fff'}">${city.name}</div>`;
      html+=`<div>Population : ${city.population}</div>`;
      html+=`<div class="tt-muted">Defense : ${city.defense||0}/${city.maxDefense||0}</div>`;
      html+=`<div class="tt-yield"><div>Nour.<span>${city.yieldFood}/t</span></div><div>Prod.<span>${city.yieldProd}/t</span></div><div>Or<span>${city.yieldGold}/t</span></div><div>Sci.<span>${city.yieldScience||0}/t</span></div></div>`;
      html+=`<div class="tt-muted">Nourriture : ${city.foodBin}/${city.foodNeeded}</div>`;
      if(city.currentBuild){
        const bn=city.currentBuild.startsWith('wonder:')
          ?'✨ '+city.currentBuild.slice(7).replace(/_/g,' ')
          :(BUILDING_FR[city.currentBuild]||city.currentBuild);
        const bc=city.currentBuild.startsWith('wonder:')?'?':(BUILDING_COSTS[city.currentBuild]||'?');
        html+=`<div class="tt-muted">Construction : ${bn} (${city.buildProgress}/${bc})</div>`;
      }
      if(city.buildings?.length) html+=`<div class="tt-muted">Batiments : ${city.buildings.map(b=>BUILDING_FR[b]||b).join(', ')}</div>`;
      if(civ?.currentResearch&&cur.techTree){
        const td=cur.techTree.find(t=>t.id===civ.currentResearch);
        if(td){const pct=Math.min(100,Math.max(0,civ.researchProg||0));html+=`<div class="tt-muted">Recherche : ${td.name} (${pct}%)</div>`;}
      }
      if(city.isCoastal) html+=`<div class="tt-muted">Ville cotiere</div>`;
    }
    if(unit){
      const civ=cur.civs[unit.civId];
      html+=`<hr class="tt-divider">`;
      html+=`<div class="tt-title" style="font-size:11px;color:${civ?.color||'#fff'}">Unite militaire</div>`;
      html+=`<div class="tt-muted">${civ?.name||'?'}</div>`;
      html+=`<div class="tt-muted">Type : ${uTypeFr(unit.type||'melee')}</div>`;
      html+=`<div class="tt-muted">Force : ${unit.strength||1}</div>`;
    }
    tooltip.innerHTML=html; tooltip.style.display='block';
  }
  let lx=(e.clientX-rect.left)+14, ly=(e.clientY-rect.top)+14;
  if(lx+230>rect.width) lx-=250;
  if(ly+220>rect.height) ly-=230;
  tooltip.style.left=lx+'px'; tooltip.style.top=ly+'px';
});
canvas.addEventListener('mouseleave',()=>{tooltip.style.display='none';});

// ── Tech tree ───────────────────────────────────────────────────────
function roundRect(c,x,y,w,h,r){
  c.beginPath(); c.moveTo(x+r,y); c.lineTo(x+w-r,y); c.quadraticCurveTo(x+w,y,x+w,y+r);
  c.lineTo(x+w,y+h-r); c.quadraticCurveTo(x+w,y+h,x+w-r,y+h);
  c.lineTo(x+r,y+h); c.quadraticCurveTo(x,y+h,x,y+h-r);
  c.lineTo(x,y+r); c.quadraticCurveTo(x,y,x+r,y);
  c.closePath(); c.fill(); c.stroke();
}
function drawTechTree(s){
  const tc=document.getElementById('techCanvas');
  if(!tc||!s.techTree) return;
  const techs=s.techTree;
  const eraGroups=[[],[],[],[]];
  for(const t of techs) if(t.era>=0&&t.era<=3) eraGroups[t.era].push(t);
  const NODE_W=110,NODE_H=30,COL_W=140,ROW_H=44,PAD_X=20,PAD_TOP=36;
  const maxRows=Math.max(...eraGroups.map(g=>g.length));
  tc.width=COL_W*4+PAD_X*2; tc.height=ROW_H*maxRows+PAD_TOP+PAD_X;
  const c=tc.getContext('2d');
  c.fillStyle='#0d1117'; c.fillRect(0,0,tc.width,tc.height);
  const pos={};
  for(let era=0;era<4;era++){
    const group=eraGroups[era];
    for(let i=0;i<group.length;i++) pos[group[i].id]={x:PAD_X+era*COL_W+COL_W/2,y:PAD_TOP+i*ROW_H+ROW_H/2};
  }
  for(let era=0;era<4;era++){
    const x=PAD_X+era*COL_W;
    c.fillStyle=ERA_COLORS[era]+'18'; c.fillRect(x,0,COL_W,tc.height);
    c.fillStyle=ERA_COLORS[era]; c.font=`bold 10px 'Share Tech Mono',monospace`;
    c.textAlign='center'; c.fillText(ERA_NAMES[era].toUpperCase(),x+COL_W/2,18);
    if(era>0){c.strokeStyle='#21293a';c.lineWidth=1;c.beginPath();c.moveTo(x,0);c.lineTo(x,tc.height);c.stroke();}
  }
  const knownByCiv={};
  for(const civ of s.civs) knownByCiv[civ.id]=new Set(civ.knownTechs||[]);
  for(const t of techs){
    const to=pos[t.id]; if(!to) continue;
    for(const req of(t.requires||[])){
      const from=pos[req]; if(!from) continue;
      c.beginPath(); c.moveTo(from.x+NODE_W/2,from.y);
      c.bezierCurveTo(from.x+NODE_W/2+30,from.y,to.x-NODE_W/2-30,to.y,to.x-NODE_W/2,to.y);
      c.strokeStyle='#2d3a50'; c.lineWidth=1.5; c.stroke();
      c.fillStyle='#2d3a50'; c.beginPath();
      c.moveTo(to.x-NODE_W/2,to.y); c.lineTo(to.x-NODE_W/2-6,to.y-3); c.lineTo(to.x-NODE_W/2-6,to.y+3); c.fill();
    }
  }
  for(const t of techs){
    const p=pos[t.id]; if(!p) continue;
    const knowers=s.civs.filter(civ=>knownByCiv[civ.id]?.has(t.id));
    const isKnown=knowers.length>0;
    const eraCol=ERA_COLORS[t.era]||'#7a8fa8';
    c.fillStyle=isKnown?eraCol+'33':'#161b24';
    c.strokeStyle=isKnown?eraCol:'#21293a';
    c.lineWidth=isKnown?2:1;
    roundRect(c,p.x-NODE_W/2,p.y-NODE_H/2,NODE_W,NODE_H,4);
    c.fillStyle=isKnown?'#fff':'#5a6a80';
    c.font=`${isKnown?'bold ':''}9px 'Share Tech Mono',monospace`;
    c.textAlign='center'; c.textBaseline='middle';
    let label=t.name;
    if(c.measureText(label).width>NODE_W-8) label=label.slice(0,14)+'...';
    c.fillText(label,p.x,p.y-(knowers.length?4:0));
    if(knowers.length){
      let dotX=p.x-(knowers.length*9)/2+4;
      for(const civ of knowers){c.beginPath();c.arc(dotX,p.y+8,3.5,0,Math.PI*2);c.fillStyle=civ.color;c.fill();dotX+=9;}
    }
    if(t.unlocksWonders?.length){c.fillStyle='#f0c06088';c.font='8px serif';c.fillText('✨',p.x+NODE_W/2-8,p.y-NODE_H/2+5);}
  }
}

// ── Boot ───────────────────────────────────────────────────────────
resize();
connect();
requestAnimationFrame(function cinematicLoop(ts){
  if (cur) {
    const sinceState = ts - lastStateTs;
    const interval = sinceState < ACTIVE_WINDOW_MS ? FX_FRAME_MS_ACTIVE : FX_FRAME_MS_IDLE;
    if (needsRender || ts - lastFrame >= interval) {
      fxNow = ts;
      lastFrame = ts;
      render(cur);
    }
  }
  requestAnimationFrame(cinematicLoop);
});
