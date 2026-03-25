package game

import (
	"fmt"
	"image"
	"image/color"
	"math/rand"
	"os"
	"sync"
	"time"
)

type Hex struct {
	Q, R     int    // Coordonnées axiales
	Terrain  string // "grassland", "ocean"
	Country  string // "Cahokia", "Washington", etc.
	EntityID int    // -1 si vide, sinon ID du Tamagotchi
}

type World struct {
	Width  int
	Height int

	Grid map[string]*Hex
	mu   sync.RWMutex
	rng  *rand.Rand
}

func NewWorld(w, h int) *World {
	source := rand.NewSource(time.Now().UnixNano())
	rng := rand.New(source)

	world := &World{
		Width:  w,
		Height: h,
		Grid:   make(map[string]*Hex),
		rng:    rng,
	}

	// Initialisation de la grille avec des cases vides
	for q := 0; q < w; q++ {
		for r := 0; r < h; r++ {
			world.Grid[serializePos(q, r)] = &Hex{
				Q:        q,
				R:        r,
				Terrain:  "ocean", // Par défaut, tout est océan
				EntityID: -1,
			}
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

func (w *World) GenerateContinent() {
	w.mu.Lock()
	defer w.mu.Unlock()

	centerQ := w.Width / 2
	centerR := w.Height / 2

	maxDistance := float64(w.Width) / 2

	for _, hex := range w.Grid {

		dq := float64(hex.Q - centerQ)
		dr := float64(hex.R - centerR)
		distance := (dq*dq + dr*dr)

		randomFactor := w.rng.Float64() * maxDistance // Ajoute de la variabilité

		if distance+randomFactor < maxDistance*maxDistance {
			hex.Terrain = "plains"

			if w.rng.Float64() < 0.15 {
				hex.Terrain = "forest"
			}
		}
	}
}

func (w *World) ApplyImageMap(path string) error {

	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return err
	}

	bounds := img.Bounds()
	imgW := bounds.Dx()
	imgH := bounds.Dy()

	w.mu.Lock()
	defer w.mu.Unlock()

	for q := 0; q < w.Width; q++ {
		for r := 0; r < w.Height; r++ {

			// Scale world coords to image coords
			x := q * (imgW - 1) / (w.Width - 1)
			y := r * (imgH - 1) / (w.Height - 1)

			pixel := img.At(x, y)
			gray := color.GrayModel.Convert(pixel).(color.Gray)

			hex := w.Grid[serializePos(q, r)]

			fmt.Println("Gray:", gray.Y)

			if gray.Y < 128 {
				hex.Terrain = "plains"
			} else {
				hex.Terrain = "ocean"
			}
		}
	}

	return nil
}

func (w *World) PrintTerrainASCII() {
	w.mu.RLock()
	defer w.mu.RUnlock()

	for r := 0; r < w.Height; r++ {
		line := make([]rune, w.Width)
		for q := 0; q < w.Width; q++ {
			hex, ok := w.Grid[serializePos(q, r)]
			if !ok {
				line[q] = '?'
				continue
			}

			switch hex.Terrain {
			case "ocean":
				line[q] = '~'
			case "forest":
				line[q] = 'T'
			default:
				line[q] = '#'
			}
		}
		fmt.Println(string(line))
	}
}
