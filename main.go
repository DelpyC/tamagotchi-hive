package main

import (
	"encoding/json"
	"log"
	"math"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// ── Constants ────────────────────────────────────────────────────────────────

const (
	tickRate = 30
	worldW   = 1400.0
	worldH   = 900.0
	tickMS   = 1000.0 / tickRate
)

// Role definitions
type Role int

const (
	RoleScout   Role = iota
	RoleSoldier Role = iota
	RoleTank    Role = iota
	RoleHealer  Role = iota
)

var roleNames = map[Role]string{
	RoleScout:   "scout",
	RoleSoldier: "soldier",
	RoleTank:    "tank",
	RoleHealer:  "healer",
}

// Role stats
type RoleStats struct {
	Radius      float64
	Speed       float64
	DPS         float64
	AttackRange float64
	HealPS      float64 // healer only
}

var roleStats = map[Role]RoleStats{
	RoleScout:   {Radius: 10, Speed: 3.5, DPS: 4, AttackRange: 30},
	RoleSoldier: {Radius: 16, Speed: 2.2, DPS: 10, AttackRange: 42},
	RoleTank:    {Radius: 26, Speed: 1.2, DPS: 6, AttackRange: 55},
	RoleHealer:  {Radius: 14, Speed: 2.0, DPS: 2, AttackRange: 35, HealPS: 8},
}

// ── Types ────────────────────────────────────────────────────────────────────

type Wall struct {
	X, Y, W, H float64
}

type MassDrop struct {
	ID    int     `json:"id"`
	X     float64 `json:"x"`
	Y     float64 `json:"y"`
	R     float64 `json:"r"`
	Color string  `json:"color"`
	TTL   int     `json:"-"` // ticks remaining
}

type Unit struct {
	ID       int     `json:"id"`
	Team     int     `json:"team"` // 0 = A (red), 1 = B (blue)
	Role     string  `json:"role"`
	X        float64 `json:"x"`
	Y        float64 `json:"y"`
	R        float64 `json:"r"`
	HP       float64 `json:"hp"`
	MaxHP    float64 `json:"maxHp"`
	Kills    int     `json:"kills"`
	VX       float64 `json:"-"`
	VY       float64 `json:"-"`
	role     Role
	speed    float64
	dps      float64
	atkRange float64
	healPS   float64
}

type TeamState struct {
	Morale float64 `json:"morale"`
	Alive  int     `json:"alive"`
	Kills  int     `json:"kills"`
}

type BattleResult struct {
	Winner   int       `json:"winner"` // 0 or 1, -1 = draw
	TeamA    TeamState `json:"teamA"`
	TeamB    TeamState `json:"teamB"`
	MVPName  string    `json:"mvpName"`
	MVPKills int       `json:"mvpKills"`
	Duration int       `json:"duration"` // ticks
}

type GameState struct {
	Units      []*Unit       `json:"units"`
	Drops      []*MassDrop   `json:"drops"`
	Walls      []Wall        `json:"walls"`
	TeamA      TeamState     `json:"teamA"`
	TeamB      TeamState     `json:"teamB"`
	Tick       int           `json:"-"`          // internal only, not sent to browser
	ElapsedSec float64       `json:"elapsedSec"` // seconds since battle started
	Phase      string        `json:"phase"`
	Result     *BattleResult `json:"result,omitempty"`
}

// ── Globals ──────────────────────────────────────────────────────────────────

var (
	state   GameState
	stateMu sync.RWMutex

	clients   = make(map[*websocket.Conn]bool)
	clientsMu sync.Mutex

	nextID int
	rng    = rand.New(rand.NewSource(time.Now().UnixNano()))

	upgrader = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}

	battleStart int
)

// ── Helpers ──────────────────────────────────────────────────────────────────

func newID() int { nextID++; return nextID }

func rndRange(lo, hi float64) float64 { return lo + rng.Float64()*(hi-lo) }

func dist(ax, ay, bx, by float64) float64 {
	return math.Sqrt((ax-bx)*(ax-bx) + (ay-by)*(ay-by))
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func normalize(dx, dy float64) (float64, float64) {
	ln := math.Sqrt(dx*dx+dy*dy) + 0.0001
	return dx / ln, dy / ln
}

// ── World generation ──────────────────────────────────────────────────────────

func buildWalls() []Wall {
	// Central chokepoint with two gaps
	cx := worldW / 2
	gap := 120.0
	thickness := 20.0
	return []Wall{
		// Top wall segment
		{cx - thickness/2, 0, thickness, worldH/2 - gap/2},
		// Bottom wall segment
		{cx - thickness/2, worldH/2 + gap/2, thickness, worldH/2 - gap/2},
		// Cover block left
		{cx - 220, worldH/2 - 40, 60, 80},
		// Cover block right
		{cx + 160, worldH/2 - 40, 60, 80},
	}
}

func collidesWall(x, y, r float64) bool {
	for _, w := range state.Walls {
		if x+r > w.X && x-r < w.X+w.W && y+r > w.Y && y-r < w.Y+w.H {
			return true
		}
	}
	return false
}

// Push unit out of walls
func resolveWall(u *Unit) {
	for _, w := range state.Walls {
		if u.X+u.R > w.X && u.X-u.R < w.X+w.W && u.Y+u.R > w.Y && u.Y-u.R < w.Y+w.H {
			// Find shortest push-out axis
			overlapL := (u.X + u.R) - w.X
			overlapR := (w.X + w.W) - (u.X - u.R)
			overlapT := (u.Y + u.R) - w.Y
			overlapB := (w.Y + w.H) - (u.Y - u.R)
			min := math.Min(math.Min(overlapL, overlapR), math.Min(overlapT, overlapB))
			switch min {
			case overlapL:
				u.X = w.X - u.R
				u.VX *= -0.5
			case overlapR:
				u.X = w.X + w.W + u.R
				u.VX *= -0.5
			case overlapT:
				u.Y = w.Y - u.R
				u.VY *= -0.5
			case overlapB:
				u.Y = w.Y + w.H + u.R
				u.VY *= -0.5
			}
		}
	}
}

// ── Unit creation ─────────────────────────────────────────────────────────────

func spawnRoster(team int, roles []Role) []*Unit {
	units := make([]*Unit, 0, len(roles))
	for i, r := range roles {
		st := roleStats[r]
		// Team 0 spawns on left, Team 1 on right
		var x, y float64
		if team == 0 {
			x = rndRange(40, worldW/2-80)
		} else {
			x = rndRange(worldW/2+80, worldW-40)
		}
		y = rndRange(40, worldH-40)
		// Ensure not inside a wall
		for collidesWall(x, y, st.Radius) {
			y = rndRange(40, worldH-40)
		}
		maxHP := st.Radius * 8
		units = append(units, &Unit{
			ID:       newID(),
			Team:     team,
			Role:     roleNames[r],
			X:        x,
			Y:        y,
			R:        st.Radius,
			HP:       maxHP,
			MaxHP:    maxHP,
			VX:       rndRange(-0.5, 0.5),
			VY:       rndRange(-0.5, 0.5),
			role:     r,
			speed:    st.Speed,
			dps:      st.DPS,
			atkRange: st.AttackRange,
			healPS:   st.HealPS,
			Kills:    0,
		})
		_ = i
	}
	return units
}

func defaultRoster(team int) []*Unit {
	roles := []Role{
		RoleScout, RoleScout, RoleScout,
		RoleSoldier, RoleSoldier, RoleSoldier, RoleSoldier, RoleSoldier,
		RoleTank, RoleTank,
		RoleHealer, RoleHealer,
	}
	return spawnRoster(team, roles)
}

// ── AI ────────────────────────────────────────────────────────────────────────

func teamMorale(team int) float64 {
	if team == 0 {
		return state.TeamA.Morale
	}
	return state.TeamB.Morale
}

func aiTarget(u *Unit) (tx, ty float64, found bool) {
	morale := teamMorale(u.Team)

	// Low morale — retreat toward own base
	if morale < 20 {
		if u.Team == 0 {
			return 80, worldH / 2, true
		}
		return worldW - 80, worldH / 2, true
	}

	enemies := make([]*Unit, 0)
	allies := make([]*Unit, 0)
	for _, other := range state.Units {
		if other.Team != u.Team {
			enemies = append(enemies, other)
		} else if other.ID != u.ID {
			allies = append(allies, other)
		}
	}

	switch u.role {

	case RoleHealer:
		// Find lowest-HP ally within a radius
		bestHP := math.MaxFloat64
		for _, a := range allies {
			if a.HP < a.MaxHP*0.7 {
				d := dist(u.X, u.Y, a.X, a.Y)
				if d < 250 && a.HP < bestHP {
					bestHP = a.HP
					tx, ty, found = a.X, a.Y, true
				}
			}
		}
		if found {
			return
		}
		// Stay slightly behind the front
		if u.Team == 0 {
			return worldW/2 - 160, worldH / 2, true
		}
		return worldW/2 + 160, worldH / 2, true

	case RoleScout:
		// Flee from any enemy that can kill it (larger role)
		for _, e := range enemies {
			d := dist(u.X, u.Y, e.X, e.Y)
			if d < e.atkRange*3 && e.R > u.R {
				dx, dy := normalize(u.X-e.X, u.Y-e.Y)
				return u.X + dx*100, u.Y + dy*100, true
			}
		}
		// Else chase weakest enemy
		fallthrough

	default:
		// 1. Am I nearly dead? flee
		if u.HP < u.MaxHP*0.15 {
			if u.Team == 0 {
				return 60, worldH / 2, true
			}
			return worldW - 60, worldH / 2, true
		}
		// 2. Attack lowest-HP enemy in range first (focus fire)
		bestScore := math.MaxFloat64
		for _, e := range enemies {
			d := dist(u.X, u.Y, e.X, e.Y)
			score := e.HP + d*0.1 // prefer low HP + closeness
			if score < bestScore {
				bestScore = score
				tx, ty, found = e.X, e.Y, true
			}
		}
		if found {
			return
		}
		// 3. Advance toward enemy base
		if u.Team == 0 {
			return worldW - 80, worldH / 2, true
		}
		return 80, worldH / 2, true
	}
}

// ── Tick ──────────────────────────────────────────────────────────────────────

func tick() {
	stateMu.Lock()
	defer stateMu.Unlock()

	if state.Phase != "battle" {
		return
	}
	state.Tick++
	dt := 1.0 / tickRate
	state.ElapsedSec = float64(state.Tick-battleStart) * dt

	// ── Move units ──
	for _, u := range state.Units {
		tx, ty, found := aiTarget(u)
		morale := teamMorale(u.Team)
		speedMod := 1.0
		if morale < 50 {
			speedMod = 0.6 + morale/50*0.4
		}

		if found {
			dx, dy := normalize(tx-u.X, ty-u.Y)
			spd := u.speed * speedMod
			u.VX = u.VX*0.75 + dx*spd*0.25
			u.VY = u.VY*0.75 + dy*spd*0.25
		}

		nx := clamp(u.X+u.VX, u.R, worldW-u.R)
		ny := clamp(u.Y+u.VY, u.R, worldH-u.R)
		u.X, u.Y = nx, ny
		resolveWall(u)

		if u.X <= u.R || u.X >= worldW-u.R {
			u.VX *= -0.8
		}
		if u.Y <= u.R || u.Y >= worldH-u.R {
			u.VY *= -0.8
		}
	}

	// ── Combat & healing ──
	dying := make(map[int]bool)
	killCredit := make(map[int]int) // unitID → killer unitID

	for i, a := range state.Units {
		if dying[a.ID] {
			continue
		}
		for j, b := range state.Units {
			if i == j || dying[b.ID] {
				continue
			}

			d := dist(a.X, a.Y, b.X, b.Y)

			// Healing (same team)
			if a.Team == b.Team && a.role == RoleHealer {
				if d < a.atkRange+b.R {
					b.HP = math.Min(b.MaxHP, b.HP+a.healPS*dt)
				}
			}

			// Attack (different team)
			if a.Team != b.Team {
				if d < a.atkRange+b.R {
					morale := teamMorale(a.Team)
					dpsMod := 1.0
					if morale < 50 {
						dpsMod = 0.5 + morale/50*0.5
					}
					b.HP -= a.dps * dpsMod * dt
					if b.HP <= 0 && !dying[b.ID] {
						dying[b.ID] = true
						killCredit[b.ID] = a.ID
					}
				}
			}
		}
	}

	// ── Process deaths ──
	teamKills := [2]int{}
	surviving := make([]*Unit, 0, len(state.Units))
	for _, u := range state.Units {
		if dying[u.ID] {
			// Drop mass
			for k := 0; k < 6; k++ {
				state.Drops = append(state.Drops, &MassDrop{
					ID:    newID(),
					X:     u.X + rndRange(-u.R, u.R),
					Y:     u.Y + rndRange(-u.R, u.R),
					R:     rndRange(3, 7),
					Color: teamColor(u.Team, true),
					TTL:   tickRate * 12,
				})
			}
			// Credit kill
			if killerID, ok := killCredit[u.ID]; ok {
				for _, k := range state.Units {
					if k.ID == killerID {
						k.Kills++
						teamKills[k.Team]++
						break
					}
				}
			}
		} else {
			surviving = append(surviving, u)
		}
	}
	state.Units = surviving
	state.TeamA.Kills += teamKills[0]
	state.TeamB.Kills += teamKills[1]

	// ── Mass drops: collect + decay ──
	liveDrop := state.Drops[:0]
	for _, drop := range state.Drops {
		drop.TTL--
		if drop.TTL <= 0 {
			continue
		}
		collected := false
		for _, u := range state.Units {
			if dist(u.X, u.Y, drop.X, drop.Y) < u.R+drop.R {
				// Small HP restore for collecting mass
				u.HP = math.Min(u.MaxHP, u.HP+2)
				collected = true
				break
			}
		}
		if !collected {
			liveDrop = append(liveDrop, drop)
		}
	}
	state.Drops = liveDrop

	// ── Morale update ──
	aliveA, aliveB := 0, 0
	for _, u := range state.Units {
		if u.Team == 0 {
			aliveA++
		} else {
			aliveB++
		}
	}
	state.TeamA.Alive = aliveA
	state.TeamB.Alive = aliveB

	// Morale decays based on losses (starts at 100)
	totalA := 12.0
	totalB := 12.0
	state.TeamA.Morale = clamp((float64(aliveA)/totalA)*100-float64(state.TeamB.Kills)*1.5, 0, 100)
	state.TeamB.Morale = clamp((float64(aliveB)/totalB)*100-float64(state.TeamA.Kills)*1.5, 0, 100)

	// ── Check victory ──
	if aliveA == 0 || aliveB == 0 {
		winner := 0
		if aliveA == 0 && aliveB > 0 {
			winner = 1
		}
		if aliveB == 0 && aliveA > 0 {
			winner = 0
		}
		if aliveA == 0 && aliveB == 0 {
			winner = -1
		}

		// Find MVP (most kills across both teams)
		// Search all original units — we track kills on survivors
		mvpKills := 0
		mvpName := "none"
		for _, u := range state.Units {
			if u.Kills > mvpKills {
				mvpKills = u.Kills
				teamLetter := "A"
				if u.Team == 1 {
					teamLetter = "B"
				}
				mvpName = teamLetter + " " + u.Role
			}
		}

		state.Phase = "result"
		state.Result = &BattleResult{
			Winner:   winner,
			TeamA:    state.TeamA,
			TeamB:    state.TeamB,
			MVPName:  mvpName,
			MVPKills: mvpKills,
			Duration: state.Tick - battleStart,
		}
	}
}

func teamColor(team int, light bool) string {
	if team == 0 {
		if light {
			return "#f09595"
		}
		return "#E24B4A"
	}
	if light {
		return "#85B7EB"
	}
	return "#378ADD"
}

// ── Broadcast ─────────────────────────────────────────────────────────────────

func broadcast() {
	stateMu.RLock()
	data, err := json.Marshal(state)
	stateMu.RUnlock()
	if err != nil {
		return
	}

	clientsMu.Lock()
	defer clientsMu.Unlock()
	for conn := range clients {
		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			conn.Close()
			delete(clients, conn)
		}
	}
}

// ── Game loop ─────────────────────────────────────────────────────────────────

func gameLoop() {
	ticker := time.NewTicker(time.Second / tickRate)
	defer ticker.Stop()
	for range ticker.C {
		tick()
		broadcast()
	}
}

// ── Init ─────────────────────────────────────────────────────────────────────

func initBattle() {
	nextID = 0
	walls := buildWalls()
	unitsA := defaultRoster(0)
	unitsB := defaultRoster(1)
	all := append(unitsA, unitsB...)
	battleStart = 0
	state = GameState{
		Units: all,
		Drops: []*MassDrop{},
		Walls: walls,
		TeamA: TeamState{Morale: 100, Alive: len(unitsA)},
		TeamB: TeamState{Morale: 100, Alive: len(unitsB)},
		Tick:  0,
		Phase: "battle",
	}
}

// ── HTTP handlers ─────────────────────────────────────────────────────────────

func wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	clientsMu.Lock()
	clients[conn] = true
	clientsMu.Unlock()

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			clientsMu.Lock()
			delete(clients, conn)
			clientsMu.Unlock()
			break
		}
		// Client can send "restart" to reset the simulation
		if string(msg) == "restart" {
			stateMu.Lock()
			initBattle()
			stateMu.Unlock()
		}
	}
}

func main() {
	initBattle()
	go gameLoop()

	http.Handle("/", http.FileServer(http.Dir("./static")))
	http.HandleFunc("/ws", wsHandler)

	log.Println("Battle server running on http://localhost:8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
