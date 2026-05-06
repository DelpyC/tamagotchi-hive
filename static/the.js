// ── Config ────────────────────────────────────────────────────────────────────
const MAP_W = 60, MAP_H = 40;
let TILE = 16; // px per tile, recalculated on resize

// Terrain colours
const TERRAIN_COLOR = {
  ocean:     '#1a3a5c',
  coast:     '#2a5a8a',
  grassland: '#3a7a3a',
  plains:    '#8a9a4a',
  desert:    '#c8a85a',
  forest:    '#1e5a2a',
  hills:     '#7a6a4a',
  mountain:  '#6a6a72',
  tundra:    '#8a9aaa',
};

// Resource icons (emoji rendered on canvas — small so no copyright concern)
const RES_ICON = {
  wheat:'🌾', cattle:'🐄', fish:'🐟',
  iron:'⚙', horses:'🐴', coal:'◼',
  gold:'★', silk:'◆', marble:'⬡',
};

// ── Canvas setup ──────────────────────────────────────────────────────────────
const canvas = document.getElementById('map');
const ctx    = canvas.getContext('2d');
const wrap   = document.getElementById('mapWrap');

function resize() {
  const maxW = Math.min(window.innerWidth - 260, 960);
  TILE = Math.floor(maxW / MAP_W);
  canvas.width  = TILE * MAP_W;
  canvas.height = TILE * MAP_H;
}
resize();
window.addEventListener('resize', resize);

// ── State ─────────────────────────────────────────────────────────────────────
let cur = null;
let prevEvents = [];

// ── WebSocket ─────────────────────────────────────────────────────────────────
const dot     = document.getElementById('dot');
const connLbl = document.getElementById('connLbl');
let ws;

function connect() {
  ws = new WebSocket(`ws://${location.host}/ws`);
  ws.onopen  = () => { dot.className = 'dot on'; connLbl.textContent = 'connected'; };
  ws.onclose = () => { dot.className = 'dot'; connLbl.textContent = 'reconnecting…'; setTimeout(connect, 2000); };
  ws.onerror = () => ws.close();
  ws.onmessage = evt => {
    prevEvents = cur ? [...cur.events] : [];
    cur = JSON.parse(evt.data);
    render(cur);
    updateSidebar(cur);
  };
}
connect();

document.getElementById('restartBtn').onclick = () => {
  if (ws && ws.readyState === 1) ws.send('restart');
};

// ── Render map ────────────────────────────────────────────────────────────────
function render(s) {
  const T = TILE;
  ctx.clearRect(0, 0, canvas.width, canvas.height);

  // Build civ color map
  const civColor = {};
  for (const c of s.civs) civColor[c.id] = c.color;

  // Build city lookup by tile
  const cityAt = {};
  for (const city of s.cities) cityAt[`${city.x},${city.y}`] = city;

  for (let y = 0; y < MAP_H; y++) {
    for (let x = 0; x < MAP_W; x++) {
      const tile = s.tiles[y][x];
      const px = x * T, py = y * T;

      // Base terrain
      ctx.fillStyle = TERRAIN_COLOR[tile.terrain] || '#333';
      ctx.fillRect(px, py, T, T);

      // Territory tint (civ color, semi-transparent)
      if (tile.civId !== -1 && civColor[tile.civId]) {
        ctx.fillStyle = civColor[tile.civId] + '33'; // ~20% alpha
        ctx.fillRect(px, py, T, T);
      }

      // Tile border (very subtle)
      ctx.strokeStyle = 'rgba(0,0,0,0.18)';
      ctx.lineWidth = 0.5;
      ctx.strokeRect(px, py, T, T);

      // Resource icon
      if (tile.resource && T >= 14) {
        const icon = RES_ICON[tile.resource];
        if (icon) {
          ctx.font = `${Math.floor(T * 0.52)}px serif`;
          ctx.textAlign = 'center';
          ctx.textBaseline = 'middle';
          ctx.fillText(icon, px + T/2, py + T/2);
        }
      }

      // City
      const city = cityAt[`${x},${y}`];
      if (city) {
        const col = civColor[city.civId] || '#fff';
        // City circle
        ctx.beginPath();
        ctx.arc(px + T/2, py + T/2, T * 0.38, 0, Math.PI*2);
        ctx.fillStyle = col;
        ctx.fill();
        ctx.strokeStyle = '#fff';
        ctx.lineWidth = 1.5;
        ctx.stroke();

        // Population number
        if (T >= 12) {
          ctx.fillStyle = '#fff';
          ctx.font = `bold ${Math.max(8, Math.floor(T * 0.42))}px sans-serif`;
          ctx.textAlign = 'center';
          ctx.textBaseline = 'middle';
          ctx.fillText(city.population, px + T/2, py + T/2);
        }

        // City name label
        console.log(TILE);
        if (T >= 14) {
          ctx.fillStyle = '#fff';
          ctx.font = `${Math.max(7, Math.floor(T * 0.38))}px 'Cinzel', serif`;
          ctx.textAlign = 'center';
          ctx.textBaseline = 'top';
          // Small shadow for readability
          ctx.shadowColor = '#000';
          ctx.shadowBlur = 3;
          ctx.fillText(city.name, px + T/2, py + T + 1);
          ctx.shadowBlur = 0;
        }
      }
    }
  }

  // Border edges — draw thicker lines where civ territory changes
  for (let y = 0; y < MAP_H; y++) {
    for (let x = 0; x < MAP_W; x++) {
      const civ = s.tiles[y][x].civId;
      if (civ === -1) continue;
      const col = (civColor[civ] || '#fff') + 'cc';
      const px = x * T, py = y * T;
      ctx.lineWidth = 1.5;
      ctx.strokeStyle = col;
      // Right edge
      if (x+1 < MAP_W && s.tiles[y][x+1].civId !== civ) {
        ctx.beginPath(); ctx.moveTo(px+T, py); ctx.lineTo(px+T, py+T); ctx.stroke();
      }
      // Bottom edge
      if (y+1 < MAP_H && s.tiles[y+1][x].civId !== civ) {
        ctx.beginPath(); ctx.moveTo(px, py+T); ctx.lineTo(px+T, py+T); ctx.stroke();
      }
      // Left edge
      if (x === 0 || s.tiles[y][x-1].civId !== civ) {
        ctx.beginPath(); ctx.moveTo(px, py); ctx.lineTo(px, py+T); ctx.stroke();
      }
      // Top edge
      if (y === 0 || s.tiles[y-1][x].civId !== civ) {
        ctx.beginPath(); ctx.moveTo(px, py); ctx.lineTo(px+T, py); ctx.stroke();
      }
    }
  }
}

// ── Sidebar ───────────────────────────────────────────────────────────────────
function updateSidebar(s) {
  document.getElementById('turnNum').textContent = s.turn;

  // Civ list
  const civList = document.getElementById('civList');
  civList.innerHTML = '';
  for (const civ of s.civs) {
    const row = document.createElement('div');
    row.className = 'civ-row';
    const cityCount = civ.cities.length;
    // Find total population
    let pop = 0;
    for (const city of s.cities) {
      if (city.civId === civ.id) pop += city.population;
    }
    row.innerHTML = `
      <div class="civ-dot" style="background:${civ.color}"></div>
      <div class="civ-name">${civ.name}</div>
      <div class="civ-stats">
        <div>🏛 ${cityCount} &nbsp; 👥 ${pop}</div>
        <div>💰 ${civ.gold} &nbsp; 🔬 ${civ.science}</div>
      </div>`;
    civList.appendChild(row);
  }

  // Event log
  const log = document.getElementById('eventLog');
  log.innerHTML = '';
  const reversed = [...s.events].reverse();
  for (let i = 0; i < reversed.length; i++) {
    const div = document.createElement('div');
    div.className = 'event' + (i === 0 ? ' new' : '');
    div.textContent = reversed[i];
    log.appendChild(div);
  }

  // Legend (only once)
  const leg = document.getElementById('legend');
  if (!leg.childElementCount) {
    for (const [name, color] of Object.entries(TERRAIN_COLOR)) {
      leg.innerHTML += `<div class="leg"><div class="leg-sq" style="background:${color}"></div>${name}</div>`;
    }
  }
}

// ── Tooltip on hover ──────────────────────────────────────────────────────────
const tooltip = document.getElementById('tooltip');

canvas.addEventListener('mousemove', e => {
  if (!cur) return;
  const rect = canvas.getBoundingClientRect();
  const mx = e.clientX - rect.left;
  const my = e.clientY - rect.top;
  const tx = Math.floor(mx / TILE);
  const ty = Math.floor(my / TILE);
  if (tx < 0 || tx >= MAP_W || ty < 0 || ty >= MAP_H) { tooltip.style.display='none'; return; }

  const tile = cur.tiles[ty][tx];
  const civColor = {};
  for (const c of cur.civs) civColor[c.id] = c.color;
  const cityAt = {};
  for (const city of cur.cities) cityAt[`${city.x},${city.y}`] = city;
  const city = cityAt[`${tx},${ty}`];

  let html = `<div class="tt-title">${tile.terrain.charAt(0).toUpperCase()+tile.terrain.slice(1)}</div>`;
  if (tile.resource) html += `<div>Resource: ${tile.resource}</div>`;
  if (tile.civId !== -1) {
    const civ = cur.civs[tile.civId];
    if (civ) html += `<div class="tt-muted">Territory of <span style="color:${civ.color}">${civ.name}</span></div>`;
  }
  if (city) {
    const civ = cur.civs[city.civId];
    html += `<hr style="border-color:#21293a;margin:5px 0">`;
    html += `<div class="tt-title" style="color:${civ?.color}">${city.name}</div>`;
    html += `<div>Population: ${city.population}</div>`;
    html += `<div>Food: ${city.yieldFood}/turn &nbsp; 🍞 ${city.foodBin}/${city.foodNeeded}</div>`;
    html += `<div>Prod: ${city.yieldProd}/turn</div>`;
    html += `<div>Gold: ${city.yieldGold}/turn</div>`;
    if (city.currentBuild) html += `<div class="tt-muted">Building: ${city.currentBuild} (${city.buildProgress}/${getBuildingCost(city.currentBuild)})</div>`;
    if (city.buildings.length) html += `<div class="tt-muted">Has: ${city.buildings.join(', ')}</div>`;
  }

  tooltip.innerHTML = html;
  tooltip.style.display = 'block';
  let lx = e.clientX - rect.left + 12;
  let ly = e.clientY - rect.top + 12;
  if (lx + 150 > canvas.width) lx -= 160;
  if (ly + 160 > canvas.height) ly -= 160;
  tooltip.style.left = lx + 'px';
  tooltip.style.top  = ly + 'px';
});

canvas.addEventListener('mouseleave', () => { tooltip.style.display='none'; });

// Building costs mirrored from Go (for tooltip display)
const BUILDING_COSTS = { granary:40, market:60, workshop:50, monument:30 };
function getBuildingCost(name) { return BUILDING_COSTS[name] || '?'; };

function renderWondersSidebar(wonders, civs) {
  const el = document.getElementById("wonders-sidebar");
  el.innerHTML = "";

  wonders.forEach(w => {
    const div = document.createElement("div");
    div.classList.add("wonder-item");

    if (w.civId === -1) {
      div.classList.add("unbuilt");
      div.innerHTML = `<strong>${w.name}</strong><br>Non construite`;
    } else {
      const civ = civs.find(c => c.id === w.civId);
      div.innerHTML = `
        <strong>${w.name}</strong><br>
        ${civ.name} (Tour ${w.turn})
      `;
    }

    el.appendChild(div);
  });
}

document.getElementById("open-tech-tree").addEventListener("click", () => {
  document.getElementById("tech-modal").classList.add("open");
  renderTechTree(cur);
});

function renderTechTree(state) {
  const canvas = document.getElementById("tech-canvas");
  const ctx = canvas.getContext("2d");

  ctx.clearRect(0, 0, canvas.width, canvas.height);

  const eraSpacing = 150;

  state.techTree.forEach((tech, i) => {
    const x = 100 + tech.era * eraSpacing;
    const y = 50 + i * 40;

    // Est-ce que au moins une civ a la tech ?
    const knownBy = state.civs.filter(c =>
      c.knownTechs.includes(tech.id)
    );

    // Node
    ctx.beginPath();
    ctx.arc(x, y, 10, 0, Math.PI * 2);

    ctx.fillStyle = knownBy.length > 0 ? "#ffd700" : "#444";
    ctx.fill();

    // Nom
    ctx.fillStyle = "#fff";
    ctx.fillText(tech.name, x + 15, y);

    // Dots par civ
    knownBy.forEach((civ, idx) => {
      ctx.fillStyle = civ.color;
      ctx.fillRect(x - 5 + idx * 4, y + 12, 3, 3);
    });

    // Flèches prerequis
    tech.requires.forEach(req => {
      const from = state.techTree.find(t => t.id === req);
      if (!from) return;

      const fx = 100 + from.era * eraSpacing;
      const fy = 50 + state.techTree.indexOf(from) * 40;

      ctx.beginPath();
      ctx.moveTo(fx, fy);
      ctx.lineTo(x, y);
      ctx.strokeStyle = "#666";
      ctx.stroke();
    });
  });
}

function attachCityTooltip(cityElement, cityData) {
  cityElement.addEventListener("mouseenter", () => {
    const tooltip = document.getElementById("tooltip");

    tooltip.innerHTML = `
      <strong>${cityData.name}</strong><br>
      Recherche : ${cityData.currentResearch.name}<br>
      Progression : ${cityData.currentResearch.progress}%
    `;

    tooltip.style.display = "block";
  });

  cityElement.addEventListener("mouseleave", () => {
    document.getElementById("tooltip").style.display = "none";
  });
}

function getResearchProgress(civ, techTree) {
  const tech = techTree.find(t => t.id === civ.currentResearch);
  if (!tech) return 0;

  return Math.floor((civ.scienceBin / tech.cost) * 100);
}

function showCityTooltip(city, civ, techTree, x, y) {
  const tooltip = document.getElementById("tooltip");

  const progress = getResearchProgress(civ, techTree);

  tooltip.innerHTML = `
    <strong>${city.name}</strong><br>
    Recherche : ${civ.currentResearch || "Aucune"}<br>
    Progression : ${progress}%
  `;

  tooltip.style.left = x + "px";
  tooltip.style.top = y + "px";
  tooltip.style.display = "block";
}