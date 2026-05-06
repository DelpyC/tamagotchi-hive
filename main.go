package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	mapW         = 60
	mapH         = 40
	numCivs      = 6
	turnInterval = 2 * time.Second
	workRadius   = 3
)

// ── Terrain ──────────────────────────────────────────────────────────────────

type TerrainType int

const (
	TerrainOcean TerrainType = iota
	TerrainCoast
	TerrainGrassland
	TerrainPlains
	TerrainDesert
	TerrainForest
	TerrainHills
	TerrainMountain
	TerrainTundra
)

var terrainNames = map[TerrainType]string{
	TerrainOcean: "ocean", TerrainCoast: "coast", TerrainGrassland: "grassland",
	TerrainPlains: "plains", TerrainDesert: "desert", TerrainForest: "forest",
	TerrainHills: "hills", TerrainMountain: "mountain", TerrainTundra: "tundra",
}

type Yield struct{ Food, Production, Gold int }

var terrainYields = map[TerrainType]Yield{
	TerrainOcean: {1, 0, 1}, TerrainCoast: {1, 0, 2}, TerrainGrassland: {2, 0, 0},
	TerrainPlains: {1, 1, 0}, TerrainDesert: {0, 0, 0}, TerrainForest: {1, 1, 0},
	TerrainHills: {0, 2, 0}, TerrainMountain: {0, 0, 0}, TerrainTundra: {1, 0, 0},
}

// ── Resources ────────────────────────────────────────────────────────────────

type ResourceType int

const (
	ResNone ResourceType = iota
	ResWheat
	ResCattle
	ResFish
	ResIron
	ResHorses
	ResCoal
	ResGold
	ResSilk
	ResMarble
)

var resourceNames = map[ResourceType]string{
	ResNone: "", ResWheat: "wheat", ResCattle: "cattle", ResFish: "fish",
	ResIron: "iron", ResHorses: "horses", ResCoal: "coal", ResGold: "gold", ResSilk: "silk", ResMarble: "marble",
}
var resourceYields = map[ResourceType]Yield{
	ResWheat: {2, 0, 0}, ResCattle: {1, 1, 0}, ResFish: {2, 0, 1},
	ResIron: {0, 2, 0}, ResHorses: {1, 1, 0}, ResCoal: {0, 2, 0},
	ResGold: {0, 0, 3}, ResSilk: {0, 0, 2}, ResMarble: {0, 1, 0},
}
var resourceTerrain = map[ResourceType][]TerrainType{
	ResWheat: {TerrainPlains, TerrainGrassland}, ResCattle: {TerrainGrassland, TerrainPlains},
	ResFish: {TerrainCoast, TerrainOcean}, ResIron: {TerrainHills, TerrainPlains},
	ResHorses: {TerrainPlains, TerrainGrassland}, ResCoal: {TerrainHills, TerrainForest},
	ResGold: {TerrainHills, TerrainDesert}, ResSilk: {TerrainForest, TerrainGrassland},
	ResMarble: {TerrainHills, TerrainPlains},
}

// ── Tech tree ─────────────────────────────────────────────────────────────────

type TechDef struct {
	ID               string   `json:"id"`
	Name             string   `json:"name"`
	Cost             int      `json:"cost"`
	Requires         []string `json:"requires"`
	Era              int      `json:"era"`
	UnlocksBuildings []string `json:"unlocksBuildings"`
	UnlocksWonders   []string `json:"unlocksWonders"`
}

var techTree = []TechDef{
	// Era 0 — Antiquité
	{ID: "agriculture", Name: "Agriculture", Cost: 20, Era: 0, Requires: []string{}, UnlocksBuildings: []string{"granary"}},
	{ID: "mining", Name: "Mines", Cost: 20, Era: 0, Requires: []string{}, UnlocksBuildings: []string{}},
	{ID: "pottery", Name: "Poterie", Cost: 25, Era: 0, Requires: []string{}, UnlocksBuildings: []string{"monument"}},
	{ID: "animal_hus", Name: "Élevage", Cost: 25, Era: 0, Requires: []string{"agriculture"}, UnlocksBuildings: []string{}},
	{ID: "sailing", Name: "Navigation", Cost: 30, Era: 0, Requires: []string{}, UnlocksBuildings: []string{"harbor"}},
	// Era 1 — Classique
	{ID: "writing", Name: "Écriture", Cost: 50, Era: 1, Requires: []string{"pottery"}, UnlocksBuildings: []string{"library"}},
	{ID: "bronze", Name: "Bronze", Cost: 55, Era: 1, Requires: []string{"mining"}, UnlocksBuildings: []string{"barracks"}},
	{ID: "masonry", Name: "Maçonnerie", Cost: 60, Era: 1, Requires: []string{"mining"}, UnlocksBuildings: []string{"walls"}, UnlocksWonders: []string{"les_pyramides"}},
	{ID: "calendar", Name: "Calendrier", Cost: 50, Era: 1, Requires: []string{"agriculture", "pottery"}, UnlocksBuildings: []string{"temple"}},
	{ID: "mathematics", Name: "Mathématiques", Cost: 70, Era: 1, Requires: []string{"writing"}, UnlocksBuildings: []string{}},
	// Era 2 — Médiéval
	{ID: "iron_working", Name: "Travail du Fer", Cost: 100, Era: 2, Requires: []string{"bronze", "mining"}, UnlocksBuildings: []string{"forge"}, UnlocksWonders: []string{"colisee"}},
	{ID: "philosophy", Name: "Philosophie", Cost: 110, Era: 2, Requires: []string{"writing", "calendar"}, UnlocksBuildings: []string{"university"}, UnlocksWonders: []string{"grande_bibliotheque"}},
	{ID: "currency", Name: "Monnaie", Cost: 100, Era: 2, Requires: []string{"mathematics"}, UnlocksBuildings: []string{"market"}},
	{ID: "construction", Name: "Construction", Cost: 110, Era: 2, Requires: []string{"masonry", "mathematics"}, UnlocksBuildings: []string{"aqueduct"}},
	{ID: "engineering", Name: "Ingénierie", Cost: 120, Era: 2, Requires: []string{"construction"}, UnlocksBuildings: []string{"workshop"}},
	// Era 3 — Renaissance
	{ID: "education", Name: "Éducation", Cost: 180, Era: 3, Requires: []string{"philosophy", "currency"}, UnlocksBuildings: []string{}},
	{ID: "astronomy", Name: "Astronomie", Cost: 190, Era: 3, Requires: []string{"education"}, UnlocksBuildings: []string{}},
	{ID: "architecture", Name: "Architecture", Cost: 200, Era: 3, Requires: []string{"engineering", "education"}, UnlocksBuildings: []string{}, UnlocksWonders: []string{"tour_eiffel", "statue_liberte"}},
	{ID: "economics", Name: "Économie", Cost: 210, Era: 3, Requires: []string{"currency", "education"}, UnlocksBuildings: []string{"bank"}},
	{ID: "gunpowder", Name: "Poudre à Canon", Cost: 220, Era: 3, Requires: []string{"iron_working", "engineering"}, UnlocksBuildings: []string{}},
}

var techByID map[string]*TechDef

func initTechIndex() {
	techByID = make(map[string]*TechDef)
	for i := range techTree {
		techByID[techTree[i].ID] = &techTree[i]
	}
}

// ── Buildings ─────────────────────────────────────────────────────────────────

type BuildingDef struct {
	Name         string
	FoodBonus    int
	ProdBonus    int
	GoldBonus    int
	ScienceBonus int
	CultureBonus int
	Cost         int
	RequiresTech string
}

var buildings = map[string]BuildingDef{
	"granary":    {"Grenier", 2, 0, 0, 0, 0, 40, "agriculture"},
	"monument":   {"Monument", 0, 0, 0, 0, 2, 30, "pottery"},
	"library":    {"Bibliothèque", 0, 0, 0, 3, 1, 60, "writing"},
	"temple":     {"Temple", 0, 0, 0, 0, 3, 55, "calendar"},
	"barracks":   {"Caserne", 0, 1, 0, 0, 0, 50, "bronze"},
	"walls":      {"Murailles", 0, 1, 0, 0, 0, 65, "masonry"},
	"market":     {"Marché", 0, 0, 3, 0, 0, 60, "currency"},
	"harbor":     {"Port", 1, 0, 2, 0, 0, 70, "sailing"},
	"forge":      {"Forge", 0, 2, 0, 0, 0, 80, "iron_working"},
	"aqueduct":   {"Aqueduc", 3, 0, 0, 0, 0, 90, "construction"},
	"workshop":   {"Atelier", 0, 2, 0, 0, 0, 75, "engineering"},
	"university": {"Université", 0, 0, 0, 5, 2, 120, "philosophy"},
	"bank":       {"Banque", 0, 0, 5, 0, 0, 110, "economics"},
}

// ── Wonders ───────────────────────────────────────────────────────────────────

type WonderDef struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	Cost         int    `json:"cost"`
	RequiresTech string `json:"requiresTech"`
	GoldBonus    int    `json:"goldBonus"`
	ScienceBonus int    `json:"scienceBonus"`
	CultureBonus int    `json:"cultureBonus"`
	FoodBonus    int    `json:"foodBonus"`
	FreeTech     bool   `json:"freeTech"`
}

var wonderDefs = map[string]WonderDef{
	"les_pyramides":       {ID: "les_pyramides", Name: "Les Pyramides", Description: "+3 nourriture dans toutes les villes, +2 culture/tour", Cost: 120, RequiresTech: "masonry", FoodBonus: 3, CultureBonus: 2},
	"colisee":             {ID: "colisee", Name: "Le Colisée", Description: "+4 culture/tour, rend les citoyens heureux", Cost: 150, RequiresTech: "iron_working", CultureBonus: 4},
	"grande_bibliotheque": {ID: "grande_bibliotheque", Name: "La Grande Bibliothèque", Description: "+6 science/tour + technologie gratuite", Cost: 200, RequiresTech: "philosophy", ScienceBonus: 6, FreeTech: true},
	"tour_eiffel":         {ID: "tour_eiffel", Name: "La Tour Eiffel", Description: "+5 culture/tour, +3 or/tour", Cost: 250, RequiresTech: "architecture", CultureBonus: 5, GoldBonus: 3},
	"statue_liberte":      {ID: "statue_liberte", Name: "La Statue de la Liberté", Description: "+4 science, +4 culture, +2 or par tour", Cost: 280, RequiresTech: "architecture", ScienceBonus: 4, CultureBonus: 4, GoldBonus: 2},
}

type WonderState struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	CivID int    `json:"civId"`
	Turn  int    `json:"turn"`
}

// ── Strategy priorities ───────────────────────────────────────────────────────

var strategyTechOrder = map[string][]string{
	"militarist":   {"mining", "bronze", "iron_working", "masonry", "engineering", "gunpowder", "agriculture", "pottery", "writing", "calendar", "mathematics", "currency", "construction", "philosophy", "education", "economics", "sailing", "architecture", "astronomy", "animal_hus"},
	"economic":     {"agriculture", "pottery", "animal_hus", "sailing", "calendar", "writing", "mathematics", "currency", "construction", "engineering", "philosophy", "education", "economics", "mining", "bronze", "masonry", "iron_working", "architecture", "astronomy", "gunpowder"},
	"expansionist": {"agriculture", "animal_hus", "pottery", "mining", "sailing", "writing", "calendar", "construction", "engineering", "masonry", "bronze", "mathematics", "currency", "philosophy", "iron_working", "education", "economics", "architecture", "astronomy", "gunpowder"},
}

var strategyBuildOrder = map[string][]string{
	"militarist":   {"barracks", "walls", "forge", "granary", "workshop", "market", "monument", "library", "temple", "harbor", "aqueduct", "university", "bank"},
	"economic":     {"market", "harbor", "granary", "bank", "aqueduct", "library", "workshop", "forge", "monument", "temple", "barracks", "walls", "university"},
	"expansionist": {"granary", "aqueduct", "monument", "temple", "market", "library", "harbor", "workshop", "forge", "barracks", "walls", "bank", "university"},
}

var strategyWonderOrder = map[string][]string{
	"militarist":   {"colisee", "les_pyramides", "grande_bibliotheque", "tour_eiffel", "statue_liberte"},
	"economic":     {"grande_bibliotheque", "tour_eiffel", "statue_liberte", "les_pyramides", "colisee"},
	"expansionist": {"les_pyramides", "statue_liberte", "grande_bibliotheque", "colisee", "tour_eiffel"},
}

// ── Tile ─────────────────────────────────────────────────────────────────────

type Tile struct {
	Terrain  string `json:"terrain"`
	Resource string `json:"resource"`
	CivID    int    `json:"civId"`
	CityID   int    `json:"cityId"`
	terrain  TerrainType
	resource ResourceType
}

// ── City ─────────────────────────────────────────────────────────────────────

type City struct {
	ID            int      `json:"id"`
	Name          string   `json:"name"`
	CivID         int      `json:"civId"`
	X             int      `json:"x"`
	Y             int      `json:"y"`
	Population    int      `json:"population"`
	FoodBin       int      `json:"foodBin"`
	FoodNeeded    int      `json:"foodNeeded"`
	CurrentBuild  string   `json:"currentBuild"`
	BuildProgress int      `json:"buildProgress"`
	Buildings     []string `json:"buildings"`
	YieldFood     int      `json:"yieldFood"`
	YieldProd     int      `json:"yieldProd"`
	YieldGold     int      `json:"yieldGold"`
	YieldScience  int      `json:"yieldScience"`
	YieldCulture  int      `json:"yieldCulture"`
	CultureAccum  int      `json:"cultureAccum"`
	BorderRadius  int      `json:"borderRadius"`
	IsCoastal     bool     `json:"isCoastal"`
}

// ── Civ ───────────────────────────────────────────────────────────────────────

type Civ struct {
	ID              int      `json:"id"`
	Name            string   `json:"name"`
	Color           string   `json:"color"`
	Gold            int      `json:"gold"`
	Science         int      `json:"science"`
	ScienceBin      int      `json:"scienceBin"`
	Culture         int      `json:"culture"`
	Cities          []int    `json:"cities"`
	Strategy        string   `json:"strategy"`
	Alive           bool     `json:"alive"`
	KnownTechs      []string `json:"knownTechs"`
	CurrentResearch string   `json:"currentResearch"`
	ResearchProg    int      `json:"researchProg"`
	Era             int      `json:"era"`
}

var civData = []struct{ name, color, strategy string }{
	{"France", "#E24B4A", "militarist"},
	{"Angleterre", "#378ADD", "economic"},
	{"Russie", "#1D9E75", "expansionist"},
	{"Espagne", "#BA7517", "expansionist"},
	{"Italie", "#7F77DD", "militarist"},
	{"Allemagne", "#D4537E", "militarist"},
}

var cityNamePool = [][]string{
	{"Moscou", "Paris", "Londres", "Madrid", "Saint-Pétersbourg", "Milan"},
	{"Barcelone", "Berlin", "Rome", "Bruxelles", "Athènes", "Kiev"},
	{"Lisbonne", "Manchester", "Birmingham", "Lyon", "Lille", "Naples"},
	{"Varsovie", "Minsk", "Vienne", "Yorkshire", "Bucarest", "Hambourg"},
	{"Turin", "Porto", "Budapest", "Glasgow", "Marseille", "Munich"},
	{"Stockholm", "Kharkiv", "Belgrade", "Zurich", "Toulouse", "Nice"},
}

// ── Game state ────────────────────────────────────────────────────────────────

type GameState struct {
	Tiles    [][]Tile      `json:"tiles"`
	Cities   []*City       `json:"cities"`
	Civs     []*Civ        `json:"civs"`
	Wonders  []WonderState `json:"wonders"`
	TechTree []TechDef     `json:"techTree"`
	Turn     int           `json:"turn"`
	Phase    string        `json:"phase"`
	Events   []string      `json:"events"`
}

// ── Globals ──────────────────────────────────────────────────────────────────

var (
	state     GameState
	stateMu   sync.RWMutex
	clients   = make(map[*websocket.Conn]bool)
	clientsMu sync.Mutex
	nextID    int
	rng       = rand.New(rand.NewSource(time.Now().UnixNano()))
	upgrader  = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
)

func newID() int { nextID++; return nextID }

// ── Helpers ───────────────────────────────────────────────────────────────────

func isLand(t TerrainType) bool {
	return t != TerrainOcean && t != TerrainCoast && t != TerrainMountain
}
func tileDist(x1, y1, x2, y2 int) float64 {
	return math.Sqrt(float64((x1-x2)*(x1-x2) + (y1-y2)*(y1-y2)))
}
func imax(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func hasTech(civ *Civ, techID string) bool {
	for _, t := range civ.KnownTechs {
		if t == techID {
			return true
		}
	}
	return false
}
func hasBuilding(city *City, name string) bool {
	for _, b := range city.Buildings {
		if b == name {
			return true
		}
	}
	return false
}
func wonderBuilt(wonderID string) bool {
	for _, w := range state.Wonders {
		if w.ID == wonderID && w.CivID != -1 {
			return true
		}
	}
	return false
}
func isCoastalCity(x, y int) bool {
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			nx, ny := x+dx, y+dy
			if nx < 0 || nx >= mapW || ny < 0 || ny >= mapH {
				continue
			}
			t := state.Tiles[ny][nx].terrain
			if t == TerrainCoast || t == TerrainOcean {
				return true
			}
		}
	}
	return false
}

// ── Noise ─────────────────────────────────────────────────────────────────────

func smoothNoise(w, h, passes int) [][]float64 {
	grid := make([][]float64, h)
	for y := range grid {
		grid[y] = make([]float64, w)
		for x := range grid[y] {
			grid[y][x] = rng.Float64()
		}
	}
	for p := 0; p < passes; p++ {
		next := make([][]float64, h)
		for y := range next {
			next[y] = make([]float64, w)
			for x := range next[y] {
				sum, cnt := 0.0, 0
				for dy := -1; dy <= 1; dy++ {
					for dx := -1; dx <= 1; dx++ {
						ny2, nx2 := y+dy, x+dx
						if ny2 >= 0 && ny2 < h && nx2 >= 0 && nx2 < w {
							sum += grid[ny2][nx2]
							cnt++
						}
					}
				}
				next[y][x] = sum / float64(cnt)
			}
		}
		grid = next
	}
	return grid
}

// ── Map ───────────────────────────────────────────────────────────────────────

func generateMap() [][]Tile {
	tiles := make([][]Tile, mapH)
	for y := range tiles {
		tiles[y] = make([]Tile, mapW)
	}
	elev := smoothNoise(mapW, mapH, 4)
	moist := smoothNoise(mapW, mapH, 3)
	temp := smoothNoise(mapW, mapH, 2)
	for y := 0; y < mapH; y++ {
		for x := 0; x < mapW; x++ {
			e, m := elev[y][x], moist[y][x]
			latFactor := math.Abs(float64(y)-float64(mapH)/2) / (float64(mapH) / 2)
			t := temp[y][x]*0.4 + (1-latFactor)*0.6
			var terrain TerrainType
			switch {
			case e < 0.30:
				terrain = TerrainOcean
			case e < 0.36:
				terrain = TerrainCoast
			case e > 0.82:
				terrain = TerrainMountain
			case e > 0.68:
				terrain = TerrainHills
			case t < 0.25:
				terrain = TerrainTundra
			case t < 0.35 && m > 0.5:
				terrain = TerrainForest
			case m < 0.3 && t > 0.6:
				terrain = TerrainDesert
			case m > 0.55:
				terrain = TerrainForest
			case m > 0.4:
				terrain = TerrainGrassland
			default:
				terrain = TerrainPlains
			}
			tiles[y][x] = Tile{Terrain: terrainNames[terrain], CivID: -1, CityID: -1, terrain: terrain}
		}
	}
	for resType, validTerrains := range resourceTerrain {
		for i := 0; i < 8+rng.Intn(6); i++ {
			for attempt := 0; attempt < 30; attempt++ {
				x, y := rng.Intn(mapW), rng.Intn(mapH)
				if tiles[y][x].resource != ResNone {
					continue
				}
				for _, vt := range validTerrains {
					if tiles[y][x].terrain == vt {
						tiles[y][x].resource = resType
						tiles[y][x].Resource = resourceNames[resType]
						break
					}
				}
				break
			}
		}
	}
	return tiles
}

// ── Yields ───────────────────────────────────────────────────────────────────

func calcYields(city *City) (food, prod, gold, science, culture int) {
	add := func(x, y int) {
		t := state.Tiles[y][x]
		ty := terrainYields[t.terrain]
		ry := resourceYields[t.resource]
		food += ty.Food + ry.Food
		prod += ty.Production + ry.Production
		gold += ty.Gold + ry.Gold
	}
	food++
	prod++
	gold++
	add(city.X, city.Y)
	worked := 0
	for radius := 1; radius <= workRadius && worked < city.Population; radius++ {
		for dy := -radius; dy <= radius; dy++ {
			for dx := -radius; dx <= radius; dx++ {
				if worked >= city.Population {
					break
				}
				if dx == 0 && dy == 0 {
					continue
				}
				nx, ny := city.X+dx, city.Y+dy
				if nx < 0 || nx >= mapW || ny < 0 || ny >= mapH {
					continue
				}
				if state.Tiles[ny][nx].CivID != city.CivID {
					continue
				}
				add(nx, ny)
				worked++
			}
		}
	}
	for _, b := range city.Buildings {
		def := buildings[b]
		food += def.FoodBonus
		prod += def.ProdBonus
		gold += def.GoldBonus
		science += def.ScienceBonus
		culture += def.CultureBonus
	}
	if !city.IsCoastal && hasBuilding(city, "harbor") {
		gold -= buildings["harbor"].GoldBonus
		food -= buildings["harbor"].FoodBonus
	}
	return
}

// ── Borders ───────────────────────────────────────────────────────────────────

func expandBorders(city *City) {
	r := city.BorderRadius
	for dy := -r; dy <= r; dy++ {
		for dx := -r; dx <= r; dx++ {
			if int(math.Sqrt(float64(dx*dx+dy*dy))) > r {
				continue
			}
			nx, ny := city.X+dx, city.Y+dy
			if nx < 0 || nx >= mapW || ny < 0 || ny >= mapH {
				continue
			}
			if state.Tiles[ny][nx].CivID == -1 {
				state.Tiles[ny][nx].CivID = city.CivID
			}
		}
	}
}

// ── Settlement ────────────────────────────────────────────────────────────────

func settlementScore(x, y, civID int) float64 {
	t := state.Tiles[y][x]
	if !isLand(t.terrain) || t.CityID != -1 {
		return -1
	}
	if t.CivID != -1 && t.CivID != civID {
		return -1
	}
	score := 0.0
	for _, city := range state.Cities {
		d := tileDist(x, y, city.X, city.Y)
		if d < 4 {
			return -1
		}
		if d < 8 {
			score -= (8 - d) * 5
		}
	}
	for dy := -2; dy <= 2; dy++ {
		for dx := -2; dx <= 2; dx++ {
			nx, ny := x+dx, y+dy
			if nx < 0 || nx >= mapW || ny < 0 || ny >= mapH {
				continue
			}
			nt := state.Tiles[ny][nx]
			ty := terrainYields[nt.terrain]
			ry := resourceYields[nt.resource]
			score += float64(ty.Food+ry.Food)*2 + float64(ty.Production+ry.Production) + float64(ty.Gold+ry.Gold)
		}
	}
	return score
}

func findBestSettlement(civID int) (int, int, bool) {
	best, bx, by := -1.0, 0, 0
	found := false
	for i := 0; i < 300; i++ {
		x, y := rng.Intn(mapW), rng.Intn(mapH)
		if s := settlementScore(x, y, civID); s > best {
			best = s
			bx, by = x, y
			found = true
		}
	}
	return bx, by, found
}

func foundCity(civID, x, y int) *City {
	civ := state.Civs[civID]
	idx := len(civ.Cities)
	if idx >= len(cityNamePool[civID]) {
		idx = len(cityNamePool[civID]) - 1
	}
	city := &City{ID: newID(), Name: cityNamePool[civID][idx], CivID: civID, X: x, Y: y,
		Population: 1, FoodNeeded: 10, Buildings: []string{}, BorderRadius: 1, IsCoastal: isCoastalCity(x, y)}
	state.Tiles[y][x].CityID = city.ID
	state.Tiles[y][x].CivID = civID
	expandBorders(city)
	civ.Cities = append(civ.Cities, city.ID)
	state.Cities = append(state.Cities, city)
	return city
}

// ── Tech AI ───────────────────────────────────────────────────────────────────

func aiChooseResearch(civ *Civ) string {
	for _, techID := range strategyTechOrder[civ.Strategy] {
		if hasTech(civ, techID) {
			continue
		}
		td := techByID[techID]
		if td == nil {
			continue
		}
		ok := true
		for _, req := range td.Requires {
			if !hasTech(civ, req) {
				ok = false
				break
			}
		}
		if ok {
			return techID
		}
	}
	return ""
}

func grantFreeTech(civ *Civ) {
	best, bestCost := "", 999999
	for _, td := range techTree {
		if hasTech(civ, td.ID) {
			continue
		}
		ok := true
		for _, req := range td.Requires {
			if !hasTech(civ, req) {
				ok = false
				break
			}
		}
		if ok && td.Cost < bestCost {
			best = td.ID
			bestCost = td.Cost
		}
	}
	if best != "" {
		civ.KnownTechs = append(civ.KnownTechs, best)
		state.Events = append(state.Events, civ.Name+" reçoit une technologie gratuite: "+techByID[best].Name)
	}
}

func civEra(civ *Civ) int {
	era := 0
	for _, tid := range civ.KnownTechs {
		td := techByID[tid]
		if td != nil && td.Era > era {
			era = td.Era
		}
	}
	return era
}

// ── Build AI ──────────────────────────────────────────────────────────────────

func aiChooseBuild(city *City) string {
	civ := state.Civs[city.CivID]
	for _, wID := range strategyWonderOrder[civ.Strategy] {
		wd, ok := wonderDefs[wID]
		if !ok || wonderBuilt(wID) || !hasTech(civ, wd.RequiresTech) {
			continue
		}
		alreadyBuilding := false
		for _, c := range state.Cities {
			if c.CivID == city.CivID && c.CurrentBuild == "wonder:"+wID {
				alreadyBuilding = true
				break
			}
		}
		if !alreadyBuilding {
			return "wonder:" + wID
		}
	}
	for _, bID := range strategyBuildOrder[civ.Strategy] {
		if hasBuilding(city, bID) {
			continue
		}
		def, ok := buildings[bID]
		if !ok {
			continue
		}
		if def.RequiresTech != "" && !hasTech(civ, def.RequiresTech) {
			continue
		}
		if bID == "harbor" && !city.IsCoastal {
			continue
		}
		return bID
	}
	return strategyBuildOrder[civ.Strategy][0]
}

// ── Turn ─────────────────────────────────────────────────────────────────────

func processTurn() {
	stateMu.Lock()
	defer stateMu.Unlock()
	state.Turn++
	if len(state.Events) > 16 {
		state.Events = state.Events[len(state.Events)-16:]
	}

	if state.Turn == 1 {
		for _, civ := range state.Civs {
			if x, y, ok := findBestSettlement(civ.ID); ok {
				city := foundCity(civ.ID, x, y)
				state.Events = append(state.Events, civ.Name+" fonde "+city.Name)
			}
		}
		return
	}

	// Research
	for _, civ := range state.Civs {
		if !civ.Alive {
			continue
		}
		if civ.CurrentResearch == "" {
			civ.CurrentResearch = aiChooseResearch(civ)
		}
		if civ.CurrentResearch == "" {
			continue
		}
		td := techByID[civ.CurrentResearch]
		if td == nil {
			civ.CurrentResearch = ""
			continue
		}
		civ.ScienceBin += civ.Science
		if civ.ScienceBin >= td.Cost {
			civ.ScienceBin -= td.Cost
			civ.KnownTechs = append(civ.KnownTechs, td.ID)
			civ.Era = civEra(civ)
			state.Events = append(state.Events, fmt.Sprintf("%s découvre %s", civ.Name, td.Name))
			civ.CurrentResearch = aiChooseResearch(civ)
		}
	}

	// Cities
	for _, city := range state.Cities {
		food, prod, gold, sci, cult := calcYields(city)
		city.YieldFood, city.YieldProd, city.YieldGold, city.YieldScience, city.YieldCulture = food, prod, gold, sci, cult
		civ := state.Civs[city.CivID]
		civ.Gold += gold
		civ.Science += sci + 1 + city.Population/3
		civ.Culture += cult + 1
		city.FoodBin += food
		if city.FoodBin >= city.FoodNeeded {
			city.FoodBin -= city.FoodNeeded
			city.Population++
			city.FoodNeeded = city.Population * 10
			state.Events = append(state.Events, fmt.Sprintf("%s: population %d", city.Name, city.Population))
		}
		city.CultureAccum += imax(1, civ.Culture/imax(1, len(civ.Cities)))
		if newR := 1 + city.CultureAccum/25; newR > city.BorderRadius && newR <= 4 {
			city.BorderRadius = newR
			for _, civ := range state.Civs {
				expandTerritory(civ.ID, len(civ.Cities))
			}
			state.Events = append(state.Events, city.Name+": frontières étendues")
		}
		if city.CurrentBuild == "" {
			city.CurrentBuild = aiChooseBuild(city)
		}
		city.BuildProgress += prod
		if len(city.CurrentBuild) > 7 && city.CurrentBuild[:7] == "wonder:" {
			wID := city.CurrentBuild[7:]
			wd := wonderDefs[wID]
			if wonderBuilt(wID) {
				city.BuildProgress = 0
				city.CurrentBuild = ""
				continue
			}
			if city.BuildProgress >= wd.Cost {
				for i := range state.Wonders {
					if state.Wonders[i].ID == wID {
						state.Wonders[i].CivID = city.CivID
						state.Wonders[i].Turn = state.Turn
						break
					}
				}
				state.Events = append(state.Events, fmt.Sprintf("✨ %s construit %s !", civ.Name, wd.Name))
				if wd.FreeTech {
					grantFreeTech(civ)
				}
				city.BuildProgress = 0
				city.CurrentBuild = ""
			}
		} else if def, ok := buildings[city.CurrentBuild]; ok && city.BuildProgress >= def.Cost {
			city.Buildings = append(city.Buildings, city.CurrentBuild)
			state.Events = append(state.Events, fmt.Sprintf("%s construit %s", city.Name, def.Name))
			city.BuildProgress = 0
			city.CurrentBuild = ""
		}
	}

	// Wonder per-turn bonuses
	for _, w := range state.Wonders {
		if w.CivID == -1 {
			continue
		}
		wd := wonderDefs[w.ID]
		civ := state.Civs[w.CivID]
		civ.Gold += wd.GoldBonus
		civ.Science += wd.ScienceBonus
		civ.Culture += wd.CultureBonus
		if wd.FoodBonus > 0 {
			for _, c := range state.Cities {
				if c.CivID == w.CivID {
					c.FoodBin += wd.FoodBonus
				}
			}
		}
	}

	// Expansion
	for _, civ := range state.Civs {
		if !civ.Alive {
			continue
		}
		if len(civ.Cities) >= 2+state.Turn/12 || civ.Gold < 15 {
			continue
		}
		if x, y, ok := findBestSettlement(civ.ID); ok {
			city := foundCity(civ.ID, x, y)
			civ.Gold -= 15
			state.Events = append(state.Events, civ.Name+" fonde "+city.Name)
		}
	}
}

// ── Init ─────────────────────────────────────────────────────────────────────

func initGame() {
	nextID = 0
	initTechIndex()
	civs := make([]*Civ, numCivs)
	for i, d := range civData {
		civs[i] = &Civ{ID: i, Name: d.name, Color: d.color, Strategy: d.strategy, Gold: 50, Cities: []int{}, Alive: true, KnownTechs: []string{}}
	}
	wonders := []WonderState{}
	for _, wd := range wonderDefs {
		wonders = append(wonders, WonderState{ID: wd.ID, Name: wd.Name, CivID: -1})
	}
	state = GameState{Tiles: generateMap(), Cities: []*City{}, Civs: civs, Wonders: wonders, TechTree: techTree, Turn: 0, Phase: "running", Events: []string{"Les civilisations s'éveillent…"}}
}

// ── Broadcast & HTTP ──────────────────────────────────────────────────────────

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

func gameLoop() {
	ticker := time.NewTicker(turnInterval)
	defer ticker.Stop()
	for range ticker.C {
		processTurn()
		broadcast()
	}
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	stateMu.RLock()
	data, _ := json.Marshal(state)
	stateMu.RUnlock()
	conn.WriteMessage(websocket.TextMessage, data)
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
		if string(msg) == "restart" {
			stateMu.Lock()
			initGame()
			stateMu.Unlock()
			broadcast()
		}
	}
}

func ownsTiles(x, y, civID int) bool {
	return state.Tiles[y][x].CivID == civID
}

func isFrontier(x, y, civID int) bool {
	dirs := [][2]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}}

	if !ownsTiles(x, y, civID) {
		return false
	}

	for _, d := range dirs {
		nx, ny := x+d[0], y+d[1]
		if nx < 0 || nx >= mapW || ny < 0 || ny >= mapH {
			continue
		}
		if state.Tiles[ny][nx].CivID != civID {
			return true // border tile
		}
	}
	return false
}

func expandTerritory(civID int, maxTiles int) int {
	frontier := [][2]int{}

	// 1. Find frontier tiles
	for y := 0; y < mapH; y++ {
		for x := 0; x < mapW; x++ {
			if isFrontier(x, y, civID) {
				frontier = append(frontier, [2]int{x, y})
			}
		}
	}

	// 2. Shuffle frontier (IMPORTANT: avoids directional bias)
	rng.Shuffle(len(frontier), func(i, j int) {
		frontier[i], frontier[j] = frontier[j], frontier[i]
	})

	dirs := [][2]int{
		{-1, 0}, {1, 0}, {0, -1}, {0, 1},
	}

	expanded := 0

	// 3. Expand from frontier
	for _, f := range frontier {
		if expanded >= maxTiles {
			break
		}

		x, y := f[0], f[1]

		// shuffle directions too (more natural spread)
		rng.Shuffle(len(dirs), func(i, j int) {
			dirs[i], dirs[j] = dirs[j], dirs[i]
		})

		for _, d := range dirs {
			nx, ny := x+d[0], y+d[1]

			if nx < 0 || nx >= mapW || ny < 0 || ny >= mapH {
				continue
			}

			tile := &state.Tiles[ny][nx]

			// Only expand into neutral land
			if tile.CivID == -1 && isLand(tile.terrain) {
				tile.CivID = civID
				expanded++
				break // move to next frontier tile
			}
		}
	}

	return expanded
}

func main() {
	initGame()
	go gameLoop()
	http.Handle("/", http.FileServer(http.Dir("./static")))
	http.HandleFunc("/ws", wsHandler)
	log.Println("Civ sim Phase 2 — http://localhost:8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
