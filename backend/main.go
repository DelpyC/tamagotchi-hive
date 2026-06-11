/*package main

import (
	"encoding/json"
	"fmt"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"net/http"
	"path/filepath"
	"tama/backend/game"
)

func main() {
	world := game.NewWorld(200, 100)
	fmt.Println("World size:", world.Width, world.Height)

	if err := world.ApplyImageMap("../Frontend/things/earth.jpg"); err != nil {
		log.Fatal("failed to apply image map:", err)
	}

	startServer(world) // ✅ pass the actual world variable
}

func startServer(myWorld *game.World) { // ✅ correct type

	frontendDir := filepath.Join("..", "Frontend")
	frontendFS := http.FileServer(http.Dir(frontendDir))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.ServeFile(w, r, filepath.Join(frontendDir, "Everything.html"))
			return
		}
		frontendFS.ServeHTTP(w, r)
	})

	http.HandleFunc("/api/map", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// If you have ToDTO()
		err := json.NewEncoder(w).Encode(myWorld.ToDTO())

		// If you DO NOT have ToDTO(), temporarily use:
		// err := json.NewEncoder(w).Encode(myWorld)

		if err != nil {
			http.Error(w, "failed to encode map", http.StatusInternalServerError)
		}
	})

	fmt.Println("Serveur lancé sur http://localhost:8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal(err)
	}
}*/

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
	tickRate   = 30     // ticks per second
	worldW     = 1200.0 // simulation world width
	worldH     = 800.0  // simulation world height
	numPellets = 150    // pellets always present in the world
	numCells   = 10     // starting cell count
	minRadius  = 12.0   // smallest a cell can be
	maxRadius  = 100.0  // largest a cell can grow
	pelletMin  = 3.0    // pellet min radius
	pelletMax  = 7.0    // pellet max radius
)

// ── Types ────────────────────────────────────────────────────────────────────

type Vec2 struct {
	X, Y float64
}

type Cell struct {
	ID    int     `json:"id"`
	X     float64 `json:"x"`
	Y     float64 `json:"y"`
	R     float64 `json:"r"`
	Color string  `json:"color"`
	VX    float64 `json:"-"`
	VY    float64 `json:"-"`
}

type Pellet struct {
	ID    int     `json:"id"`
	X     float64 `json:"x"`
	Y     float64 `json:"y"`
	R     float64 `json:"r"`
	Color string  `json:"color"`
}

type GameState struct {
	Cells   []*Cell   `json:"cells"`
	Pellets []*Pellet `json:"pellets"`
	Tick    int       `json:"tick"`
}

// ── Globals ──────────────────────────────────────────────────────────────────

var (
	cellColors   = []string{"#7F77DD", "#1D9E75", "#D85A30", "#D4537E", "#378ADD", "#639922", "#BA7517", "#E24B4A", "#534AB7", "#0F6E56"}
	pelletColors = []string{"#9FE1CB", "#AFA9EC", "#F0997B", "#85B7EB", "#C0DD97", "#FAC775"}

	state   GameState
	stateMu sync.RWMutex

	clients   = make(map[*websocket.Conn]bool)
	clientsMu sync.Mutex

	nextID int
	rng    = rand.New(rand.NewSource(time.Now().UnixNano()))

	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
)

// ── Helpers ──────────────────────────────────────────────────────────────────

func newID() int {
	nextID++
	return nextID
}

func rndRange(lo, hi float64) float64 {
	return lo + rng.Float64()*(hi-lo)
}

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

// ── Initialisation ───────────────────────────────────────────────────────────

func newCell(i int) *Cell {
	r := rndRange(minRadius, minRadius+20)
	return &Cell{
		ID:    newID(),
		X:     rndRange(r, worldW-r),
		Y:     rndRange(r, worldH-r),
		R:     r,
		Color: cellColors[i%len(cellColors)],
		VX:    rndRange(-1, 1),
		VY:    rndRange(-1, 1),
	}
}

func newPellet() *Pellet {
	r := rndRange(pelletMin, pelletMax)
	return &Pellet{
		ID:    newID(),
		X:     rndRange(5, worldW-5),
		Y:     rndRange(5, worldH-5),
		R:     r,
		Color: pelletColors[rng.Intn(len(pelletColors))],
	}
}

func initState() {
	state.Cells = make([]*Cell, numCells)
	for i := range state.Cells {
		state.Cells[i] = newCell(i)
	}
	state.Pellets = make([]*Pellet, numPellets)
	for i := range state.Pellets {
		state.Pellets[i] = newPellet()
	}
}

// ── AI steering ──────────────────────────────────────────────────────────────

// findTarget returns the position the cell wants to move toward.
// Priority: flee from bigger cells, else chase nearest prey (small cell or pellet).
func findTarget(cell *Cell) (tx, ty float64, found bool) {
	// 1. Flee from larger cells nearby
	for _, other := range state.Cells {
		if other.ID == cell.ID {
			continue
		}
		if other.R > cell.R*1.15 {
			d := dist(cell.X, cell.Y, other.X, other.Y)
			if d < other.R*5 {
				// run directly away
				dx := cell.X - other.X
				dy := cell.Y - other.Y
				ln := math.Sqrt(dx*dx+dy*dy) + 0.001
				return cell.X + dx/ln*50, cell.Y + dy/ln*50, true
			}
		}
	}

	// 2. Chase nearest smaller cell
	bestDist := math.MaxFloat64
	for _, other := range state.Cells {
		if other.ID == cell.ID {
			continue
		}
		if cell.R > other.R*1.15 {
			d := dist(cell.X, cell.Y, other.X, other.Y)
			if d < bestDist {
				bestDist = d
				tx, ty, found = other.X, other.Y, true
			}
		}
	}

	// 3. Chase nearest pellet (always fallback)
	for _, p := range state.Pellets {
		d := dist(cell.X, cell.Y, p.X, p.Y)
		if d < bestDist {
			bestDist = d
			tx, ty, found = p.X, p.Y, true
		}
	}

	return
}

// ── Simulation tick ───────────────────────────────────────────────────────────

func tick() {
	stateMu.Lock()
	defer stateMu.Unlock()

	state.Tick++

	// Move cells
	for _, cell := range state.Cells {
		tx, ty, found := findTarget(cell)
		if found {
			dx := tx - cell.X
			dy := ty - cell.Y
			ln := math.Sqrt(dx*dx+dy*dy) + 0.001
			// Larger cells are slower
			spd := (15.0 / cell.R) * 1.5
			cell.VX = cell.VX*0.8 + (dx/ln)*spd*0.2
			cell.VY = cell.VY*0.8 + (dy/ln)*spd*0.2
		}

		cell.X = clamp(cell.X+cell.VX, cell.R, worldW-cell.R)
		cell.Y = clamp(cell.Y+cell.VY, cell.R, worldH-cell.R)

		// Bounce off walls
		if cell.X <= cell.R || cell.X >= worldW-cell.R {
			cell.VX *= -1
		}
		if cell.Y <= cell.R || cell.Y >= worldH-cell.R {
			cell.VY *= -1
		}
	}

	// Cell eats pellet
	newPellets := state.Pellets[:0]
	for _, p := range state.Pellets {
		eaten := false
		for _, cell := range state.Cells {
			if dist(cell.X, cell.Y, p.X, p.Y) < cell.R+p.R*0.5 {
				cell.R = math.Min(maxRadius, math.Sqrt(cell.R*cell.R+p.R*p.R*0.3))
				eaten = true
				break
			}
		}
		if !eaten {
			newPellets = append(newPellets, p)
		}
	}
	// Replenish eaten pellets
	for len(newPellets) < numPellets {
		newPellets = append(newPellets, newPellet())
	}
	state.Pellets = newPellets

	// Cell eats cell
	surviving := make([]*Cell, 0, len(state.Cells))
	eaten := make(map[int]bool)
	for i, a := range state.Cells {
		if eaten[a.ID] {
			continue
		}
		for j, b := range state.Cells {
			if i == j || eaten[b.ID] || eaten[a.ID] {
				continue
			}
			if a.R > b.R*1.1 && dist(a.X, a.Y, b.X, b.Y) < a.R*0.9 {
				a.R = math.Min(maxRadius, math.Sqrt(a.R*a.R+b.R*b.R*0.85))
				eaten[b.ID] = true
			}
		}
		if !eaten[a.ID] {
			surviving = append(surviving, a)
		}
	}

	// Respawn eaten cells so the world stays populated
	for len(surviving) < numCells {
		surviving = append(surviving, newCell(len(surviving)))
	}
	state.Cells = surviving

	// Gradually shrink large cells (mass decay) to keep game dynamic
	for _, cell := range state.Cells {
		if cell.R > minRadius*2 {
			cell.R -= 0.02
		}
	}
}

// ── WebSocket broadcast ───────────────────────────────────────────────────────

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

// ── Game loop ────────────────────────────────────────────────────────────────

func gameLoop() {
	ticker := time.NewTicker(time.Second / tickRate)
	defer ticker.Stop()
	for range ticker.C {
		tick()
		broadcast()
	}
}

// ── HTTP handlers ─────────────────────────────────────────────────────────────

func wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WS upgrade error:", err)
		return
	}
	clientsMu.Lock()
	clients[conn] = true
	clientsMu.Unlock()
	log.Println("Client connected:", conn.RemoteAddr())

	// Keep reading to detect disconnects (we don't expect messages from client)
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			clientsMu.Lock()
			delete(clients, conn)
			clientsMu.Unlock()
			log.Println("Client disconnected:", conn.RemoteAddr())
			break
		}
	}
}

// ── Entry point ───────────────────────────────────────────────────────────────

func main() {
	initState()
	go gameLoop()

	http.Handle("/", http.FileServer(http.Dir("./static")))
	http.HandleFunc("/ws", wsHandler)

	log.Println("Server running on http://localhost:8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
