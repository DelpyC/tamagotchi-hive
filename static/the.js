// ── Canvas setup ──────────────────────────────────────────────────────────────
const canvas = document.getElementById('canvas');
const ctx = canvas.getContext('2d');
const WORLD_W = 1400, WORLD_H = 900;

function resize() {
    const maxW = Math.min(window.innerWidth - 32, 1200);
    const scale = maxW / WORLD_W;
    canvas.width = Math.round(WORLD_W * scale);
    canvas.height = Math.round(WORLD_H * scale);
    canvas.dataset.scale = scale;
}
resize();
window.addEventListener('resize', resize);

// ── State ─────────────────────────────────────────────────────────────────────
let cur = null, prev = null, lastTick = 0;
const TICK_MS = 1000 / 30;

// Role display chars
const roleChar = { scout: 'S', soldier: 'M', tank: 'T', healer: 'H' };

// ── WebSocket ─────────────────────────────────────────────────────────────────
let ws;
const connDot = document.getElementById('connDot');
const connLabel = document.getElementById('connLabel');

function connect() {
    ws = new WebSocket(`ws://${location.host}/ws`);

    ws.onopen = () => {
        connDot.className = 'on';
        connLabel.textContent = 'connected';
    };

    ws.onmessage = evt => {
        prev = cur;
        cur = JSON.parse(evt.data);
        lastTick = performance.now();
        updateHUD(cur);
        if (cur.phase === 'result') showResult(cur.result);
    };

    ws.onclose = () => {
        connDot.className = '';
        connLabel.textContent = 'reconnecting…';
        setTimeout(connect, 2000);
    };
    ws.onerror = () => ws.close();
}
connect();

document.getElementById('restartBtn').onclick = () => {
    document.getElementById('resultOverlay').classList.remove('show');
    if (ws && ws.readyState === 1) ws.send('restart');
};

// ── HUD update ────────────────────────────────────────────────────────────────
function updateHUD(s) {
    document.getElementById('tickDisp').textContent = s.tick;
    document.getElementById('phaseLabel').textContent = s.phase.toUpperCase();
    document.getElementById('aliveA').textContent = s.teamA.alive;
    document.getElementById('killsA').textContent = s.teamA.kills;
    document.getElementById('moraleA').textContent = Math.round(s.teamA.morale);
    document.getElementById('moraleBarA').style.width = s.teamA.morale + '%';
    document.getElementById('aliveB').textContent = s.teamB.alive;
    document.getElementById('killsB').textContent = s.teamB.kills;
    document.getElementById('moraleB').textContent = Math.round(s.teamB.morale);
    document.getElementById('moraleBarB').style.width = s.teamB.morale + '%';
}

function showResult(r) {
    const overlay = document.getElementById('resultOverlay');
    const title = document.getElementById('resultTitle');
    const sub = document.getElementById('resultSub');
    overlay.classList.add('show');
    if (r.winner === -1) {
        title.textContent = 'DRAW'; title.className = 'draw';
        sub.textContent = 'MUTUAL ANNIHILATION';
    } else if (r.winner === 0) {
        title.textContent = 'VICTORY'; title.className = 'red';
        sub.textContent = 'TEAM ALPHA WINS';
    } else {
        title.textContent = 'VICTORY'; title.className = 'blue';
        sub.textContent = 'TEAM BRAVO WINS';
    }
    document.getElementById('rKillsA').textContent = r.teamA.kills;
    document.getElementById('rKillsB').textContent = r.teamB.kills;
    document.getElementById('rMoraleA').textContent = Math.round(r.teamA.morale);
    document.getElementById('rMoraleB').textContent = Math.round(r.teamB.morale);
    document.getElementById('rDuration').textContent = r.duration;
    document.getElementById('rMVP').textContent = r.mvpName;
    document.getElementById('rMVPKills').textContent = r.mvpKills;
}

// ── Interpolation ────────────────────────────────────────────────────────────
function lerp(a, b, t) { return a + (b - a) * t; }

function interpUnits(p, c, t) {
    if (!p) return c.units;
    const pmap = {};
    for (const u of p.units) pmap[u.id] = u;
    return c.units.map(u => {
        const pu = pmap[u.id];
        if (!pu) return u;
        return { ...u, x: lerp(pu.x, u.x, t), y: lerp(pu.y, u.y, t) };
    });
}

// ── Render ────────────────────────────────────────────────────────────────────
const RED = '#E24B4A', RED_L = '#f09595';
const BLUE = '#378ADD', BLUE_L = '#85B7EB';
const WALL_COLOR = '#1e2430';
const WALL_STROKE = '#2d3547';
const DROP_A = '#f09595', DROP_B = '#85B7EB';

function teamColor(team, light) {
    return team === 0 ? (light ? RED_L : RED) : (light ? BLUE_L : BLUE);
}

function draw() {
    requestAnimationFrame(draw);
    if (!cur) return;

    const scale = parseFloat(canvas.dataset.scale) || 1;
    const W = canvas.width, H = canvas.height;
    const t = Math.min((performance.now() - lastTick) / TICK_MS, 1);
    const units = interpUnits(prev, cur, t);

    // Background
    ctx.fillStyle = '#0a0b0e';
    ctx.fillRect(0, 0, W, H);

    // Grid
    ctx.strokeStyle = 'rgba(255,255,255,0.025)';
    ctx.lineWidth = 0.5;
    const g = 50 * scale;
    for (let x = 0; x < W; x += g) { ctx.beginPath(); ctx.moveTo(x, 0); ctx.lineTo(x, H); ctx.stroke(); }
    for (let y = 0; y < H; y += g) { ctx.beginPath(); ctx.moveTo(0, y); ctx.lineTo(W, y); ctx.stroke(); }

    // Team territory tint
    ctx.fillStyle = 'rgba(226,75,74,0.03)';
    ctx.fillRect(0, 0, W / 2, H);
    ctx.fillStyle = 'rgba(55,138,221,0.03)';
    ctx.fillRect(W / 2, 0, W / 2, H);

    // Centre line
    ctx.setLineDash([6, 6]);
    ctx.strokeStyle = 'rgba(255,255,255,0.06)';
    ctx.lineWidth = 1;
    ctx.beginPath(); ctx.moveTo(W / 2, 0); ctx.lineTo(W / 2, H); ctx.stroke();
    ctx.setLineDash([]);

    // Walls
    for (const w of cur.walls) {
        const wx = w.X * scale, wy = w.Y * scale, ww = w.W * scale, wh = w.H * scale;
        ctx.fillStyle = WALL_COLOR;
        ctx.fillRect(wx, wy, ww, wh);
        ctx.strokeStyle = WALL_STROKE;
        ctx.lineWidth = 1;
        ctx.strokeRect(wx, wy, ww, wh);
    }

    // Mass drops
    for (const d of cur.drops) {
        ctx.beginPath();
        ctx.arc(d.x * scale, d.y * scale, d.r * scale, 0, Math.PI * 2);
        ctx.fillStyle = d.color + '99';
        ctx.fill();
    }

    // Attack range rings (faint, only for units in combat)
    // (skipped for perf — add if desired)

    // Units
    const sorted = [...units].sort((a, b) => a.r - b.r);
    for (const u of sorted) {
        const cx = u.x * scale, cy = u.y * scale, cr = u.r * scale;
        const col = teamColor(u.team, false);
        const colL = teamColor(u.team, true);

        // HP arc background
        ctx.beginPath();
        ctx.arc(cx, cy, cr + 3 * scale, 0, Math.PI * 2);
        ctx.strokeStyle = 'rgba(255,255,255,0.06)';
        ctx.lineWidth = 2.5 * scale;
        ctx.stroke();

        // HP arc fill
        const hpFrac = Math.max(0, u.hp / u.maxHp);
        ctx.beginPath();
        ctx.arc(cx, cy, cr + 3 * scale, -Math.PI / 2, -Math.PI / 2 + hpFrac * Math.PI * 2);
        ctx.strokeStyle = hpFrac > 0.5 ? col : '#E24B4A';
        ctx.lineWidth = 2.5 * scale;
        ctx.stroke();

        // Body
        ctx.beginPath();
        ctx.arc(cx, cy, cr, 0, Math.PI * 2);
        ctx.fillStyle = col + 'cc';
        ctx.fill();
        ctx.strokeStyle = col;
        ctx.lineWidth = 1.5;
        ctx.stroke();

        // Role letter
        const fs = Math.max(8, cr * 0.7);
        ctx.font = `${fs}px 'Share Tech Mono', monospace`;
        ctx.textAlign = 'center';
        ctx.textBaseline = 'middle';
        ctx.fillStyle = '#fff';
        ctx.fillText(roleChar[u.role] || '?', cx, cy);

        // Kill badge
        if (u.kills > 0 && cr > 8) {
            const bx = cx + cr * 0.7, by = cy - cr * 0.7;
            ctx.beginPath();
            ctx.arc(bx, by, 5 * scale, 0, Math.PI * 2);
            ctx.fillStyle = '#f5c542';
            ctx.fill();
            ctx.font = `${Math.max(6, 5 * scale)}px 'Share Tech Mono', monospace`;
            ctx.fillStyle = '#000';
            ctx.fillText(u.kills, bx, by);
        }

        // Low morale flee indicator
        if (u.hp < u.maxHp * 0.15) {
            ctx.font = `${Math.max(8, cr * 0.5)}px monospace`;
            ctx.fillStyle = '#f5c542';
            ctx.fillText('!', cx, cy - cr - 6 * scale);
        }
    }
}

draw();
