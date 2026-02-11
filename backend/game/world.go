package game

import (
	"fmt"
	"sync"
)

type Hex struct {
	Q        int    `json:"q"` // Coordonnées axiales
	R        int    `json:"r"`
	Terrain  string `json:"Terrain"`  // "grassland", "ocean"
	Country  string `json:"Country"`  // "Cahokia", "Washington", etc.
	EntityID int    `json:"EntityID"` // -1 si vide, sinon ID du Tamagotchi
}

type World struct {
	Width  int `json:"width"`
	Height int `json:"height"`

	Grid map[string]*Hex `json:"grid"`
	mu   sync.RWMutex
}

func NewWorld(w, h int) *World {
	world := &World{
		Width:  w,
		Height: h,
		Grid:   make(map[string]*Hex),
	}

	// Initialisation de la grille avec des cases vides
	for q := 0; q < w; q++ {
		for r := 0; r < h; r++ {
			hex := &Hex{
				Q:        q,
				R:        r,
				Terrain:  "ocean", // Par défaut
				EntityID: -1,
			}
			world.Grid[serializePos(q, r)] = hex
		}
	}
	return world
}
func (w *World) IsFree(q, r int) bool {
	w.mu.RLock()
	defer w.mu.RUnlock()

	// 1. Vérification des limites de la grille
	if q < 0 || q >= w.Width || r < 0 || r >= w.Height {
		return false
	}

	// 2. Récupération de la case dans la map
	hex, exists := w.Grid[serializePos(q, r)]
	if !exists {
		return false
	}

	// 3. La case est libre uniquement si EntityID est à -1
	// On a supprimé la vérification du terrain "mountain"
	return hex.EntityID == -1
}
func serializePos(q, r int) string {
	return fmt.Sprintf("%d,%d", q, r)
}
func (w *World) GetCells() ([]byte, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	cells := make([]*Hex, 0, len(w.Grid))
	for _, hex := range w.Grid {
		cells = append(cells, hex)
	}
	return json.Marshal(cells)
}
