package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net/http"
	"strings"
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
	maxTurns     = 280
	settlerCost  = 70
)

// Terrain

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

// Resources

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

// Tech tree

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
	// Era 0
	{ID: "agriculture", Name: "Agriculture", Cost: 20, Era: 0, Requires: []string{}, UnlocksBuildings: []string{"granary"}},
	{ID: "mining", Name: "Mines", Cost: 20, Era: 0, Requires: []string{}, UnlocksBuildings: []string{}},
	{ID: "pottery", Name: "Poterie", Cost: 25, Era: 0, Requires: []string{}, UnlocksBuildings: []string{"monument"}},
	{ID: "animal_hus", Name: "Elevage", Cost: 25, Era: 0, Requires: []string{"agriculture"}, UnlocksBuildings: []string{}},
	{ID: "sailing", Name: "Navigation", Cost: 30, Era: 0, Requires: []string{}, UnlocksBuildings: []string{"harbor"}},
	// Era 1
	{ID: "writing", Name: "Ecriture", Cost: 50, Era: 1, Requires: []string{"pottery"}, UnlocksBuildings: []string{"library"}},
	{ID: "bronze", Name: "Bronze", Cost: 55, Era: 1, Requires: []string{"mining"}, UnlocksBuildings: []string{"barracks"}},
	{ID: "masonry", Name: "Maconnerie", Cost: 60, Era: 1, Requires: []string{"mining"}, UnlocksBuildings: []string{"walls"}, UnlocksWonders: []string{"pyramids"}},
	{ID: "calendar", Name: "Calendrier", Cost: 50, Era: 1, Requires: []string{"agriculture", "pottery"}, UnlocksBuildings: []string{"temple"}},
	{ID: "mathematics", Name: "Mathematiques", Cost: 70, Era: 1, Requires: []string{"writing"}, UnlocksBuildings: []string{}},
	// Era 2
	{ID: "iron_working", Name: "Travail du Fer", Cost: 100, Era: 2, Requires: []string{"bronze", "mining"}, UnlocksBuildings: []string{"forge"}, UnlocksWonders: []string{"colosseum"}},
	{ID: "philosophy", Name: "Philosophie", Cost: 110, Era: 2, Requires: []string{"writing", "calendar"}, UnlocksBuildings: []string{"university"}, UnlocksWonders: []string{"oxford_university"}},
	{ID: "currency", Name: "Monnaie", Cost: 100, Era: 2, Requires: []string{"mathematics"}, UnlocksBuildings: []string{"market"}},
	{ID: "construction", Name: "Construction", Cost: 110, Era: 2, Requires: []string{"masonry", "mathematics"}, UnlocksBuildings: []string{"aqueduct"}},
	{ID: "engineering", Name: "Ingenierie", Cost: 120, Era: 2, Requires: []string{"construction"}, UnlocksBuildings: []string{"workshop"}},
	// Era 3
	{ID: "education", Name: "Education", Cost: 180, Era: 3, Requires: []string{"philosophy", "currency"}, UnlocksBuildings: []string{}},
	{ID: "astronomy", Name: "Astronomie", Cost: 190, Era: 3, Requires: []string{"education"}, UnlocksBuildings: []string{}},
	{ID: "architecture", Name: "Architecture", Cost: 200, Era: 3, Requires: []string{"engineering", "education"}, UnlocksBuildings: []string{}, UnlocksWonders: []string{"eiffel_tower", "statue_of_liberty"}},
	{ID: "economics", Name: "Economie", Cost: 210, Era: 3, Requires: []string{"currency", "education"}, UnlocksBuildings: []string{"bank"}},
	{ID: "gunpowder", Name: "Poudre a Canon", Cost: 220, Era: 3, Requires: []string{"iron_working", "engineering"}, UnlocksBuildings: []string{}},
}

var techByID map[string]*TechDef

func initTechIndex() {
	techByID = make(map[string]*TechDef)
	for i := range techTree {
		techByID[techTree[i].ID] = &techTree[i]
	}
}

// Buildings

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
	"library":    {"Bibliotheque", 0, 0, 0, 3, 1, 60, "writing"},
	"temple":     {"Temple", 0, 0, 0, 0, 3, 55, "calendar"},
	"barracks":   {"Caserne", 0, 1, 0, 0, 0, 50, "bronze"},
	"walls":      {"Murailles", 0, 1, 0, 0, 0, 65, "masonry"},
	"market":     {"Marche", 0, 0, 3, 0, 0, 60, "currency"},
	"harbor":     {"Port", 1, 0, 2, 0, 0, 70, "sailing"},
	"forge":      {"Forge", 0, 2, 0, 0, 0, 80, "iron_working"},
	"aqueduct":   {"Aqueduc", 3, 0, 0, 0, 0, 90, "construction"},
	"workshop":   {"Atelier", 0, 2, 0, 0, 0, 75, "engineering"},
	"university": {"Universite", 0, 0, 0, 5, 2, 120, "philosophy"},
	"bank":       {"Banque", 0, 0, 5, 0, 0, 110, "economics"},
}

// Wonders

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
	"pyramids":          {ID: "pyramids", Name: "Pyramides", Description: "+3 nourriture dans toutes les villes, +2 culture/tour", Cost: 120, RequiresTech: "masonry", FoodBonus: 3, CultureBonus: 2},
	"colosseum":         {ID: "colosseum", Name: "Colisee", Description: "+4 culture/tour, rend les citoyens heureux", Cost: 150, RequiresTech: "iron_working", CultureBonus: 4},
	"oxford_university": {ID: "oxford_university", Name: "Universite d'Oxford", Description: "+6 science/tour + technologie gratuite", Cost: 200, RequiresTech: "philosophy", ScienceBonus: 6, FreeTech: true},
	"eiffel_tower":      {ID: "eiffel_tower", Name: "Tour Eiffel", Description: "+5 culture/tour, +3 or/tour", Cost: 250, RequiresTech: "architecture", CultureBonus: 5, GoldBonus: 3},
	"statue_of_liberty": {ID: "statue_of_liberty", Name: "Statue de la Liberte", Description: "+4 science, +4 culture, +2 or par tour", Cost: 280, RequiresTech: "architecture", ScienceBonus: 4, CultureBonus: 4, GoldBonus: 2},
}

type WonderState struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	CivID  int    `json:"civId"`
	CityID int    `json:"cityId"`
	Turn   int    `json:"turn"`
}

// Strategy priorities

var strategyTechOrder = map[string][]string{
	"militarist":   {"mining", "bronze", "iron_working", "masonry", "engineering", "gunpowder", "agriculture", "pottery", "writing", "calendar", "mathematics", "currency", "construction", "philosophy", "education", "economics", "sailing", "architecture", "astronomy", "animal_hus"},
	"economic":     {"agriculture", "pottery", "animal_hus", "sailing", "calendar", "writing", "mathematics", "currency", "construction", "engineering", "philosophy", "education", "economics", "mining", "bronze", "masonry", "iron_working", "architecture", "astronomy", "gunpowder"},
	"expansionist": {"agriculture", "animal_hus", "pottery", "mining", "sailing", "writing", "calendar", "construction", "engineering", "masonry", "bronze", "mathematics", "currency", "philosophy", "iron_working", "education", "economics", "architecture", "astronomy", "gunpowder"},
}

var strategyBuildOrder = map[string][]string{
	"militarist":   {"barracks", "walls", "settler", "forge", "granary", "workshop", "market", "monument", "library", "temple", "harbor", "aqueduct", "university", "bank"},
	"economic":     {"market", "harbor", "settler", "granary", "bank", "aqueduct", "library", "workshop", "forge", "monument", "temple", "barracks", "walls", "university"},
	"expansionist": {"settler", "granary", "aqueduct", "monument", "temple", "market", "library", "harbor", "workshop", "forge", "barracks", "walls", "bank", "university"},
}

var strategyWonderOrder = map[string][]string{
	"militarist":   {"colosseum", "pyramids", "oxford_university", "eiffel_tower", "statue_of_liberty"},
	"economic":     {"oxford_university", "eiffel_tower", "statue_of_liberty", "pyramids", "colosseum"},
	"expansionist": {"pyramids", "statue_of_liberty", "oxford_university", "colosseum", "eiffel_tower"},
}

// Tile

type Tile struct {
	Terrain  string `json:"terrain"`
	Resource string `json:"resource"`
	CivID    int    `json:"civId"`
	CityID   int    `json:"cityId"`
	Explored bool   `json:"explored"`
	Visible  bool   `json:"visible"`
	terrain  TerrainType
	resource ResourceType
}

// City

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
	Defense       int      `json:"defense"`
	MaxDefense    int      `json:"maxDefense"`
}

// Civ

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

type Unit struct {
	ID       int    `json:"id"`
	CivID    int    `json:"civId"`
	X        int    `json:"x"`
	Y        int    `json:"y"`
	Strength int    `json:"strength"`
	Type     string `json:"type"`
}

var cityNamePool = [][]string{
	{"Rome", "Antium", "Cumae", "Neapolis", "Capua", "Ravenna"},
	{"Athenes", "Sparte", "Corinthe", "Argos", "Cnossos", "Pharsale"},
	{"Persepolis", "Suse", "Pasargades", "Ecbatane", "Bactra", "Sardes"},
	{"Le Caire", "Memphis", "Thebes", "Alexandrie", "Heliopolis", "Louxor"},
	{"Oslo", "Bergen", "Trondheim", "Uppsala", "Hedeby", "Ribe"},
	{"Karakorum", "Samarkand", "Tabriz", "Khanbaliq", "Beshbalik", "Almaliq"},
}

// Game state

var territoryData = []struct{ name, color, strategy string }{
	{"France", "#0055A4", "economic"},
	{"Italia", "#008C45", "economic"},
	{"Deutschland", "#2B2F3A", "militarist"},
	{"Espana", "#AA151B", "expansionist"},
	{"United-Kingdom", "#ededed", "militarist"},
	{"Belgique", "#FFD700", "economic"},
}

var territoryCityNamePool = [][]string{
	{"Paris", "Lyon", "Marseille", "Toulouse", "Bordeaux", "Lille", "Nice", "Nantes", "Strasbourg", "Montpellier", "Rennes", "Reims", "Le Havre", "Saint-Etienne", "Toulon", "Grenoble", "Dijon", "Angers", "Nimes", "Villeurbanne", "Clermont-Ferrand", "Amiens", "Metz", "Besancon"},
	{"Roma", "Milano", "Napoli", "Torino", "Firenze", "Venezia", "Palermo", "Genova", "Bologna", "Pisa", "Verona", "Bari", "Catania", "Padova", "Trieste", "Taranto", "Brescia", "Parma", "Modena", "Perugia", "Livorno", "Ravenna", "Ferrara", "Siena"},
	{"Berlin", "Hamburg", "Munchen", "Koln", "Frankfurt", "Dresden", "Stuttgart", "Dortmund", "Essen", "Leipzig", "Bremen", "Hannover", "Nurnberg", "Duisburg", "Bochum", "Wuppertal", "Bielefeld", "Bonn", "Munster", "Karlsruhe", "Mannheim", "Augsburg", "Wiesbaden", "Kiel"},
	{"Madrid", "Barcelona", "Valencia", "Sevilla", "Bilbao", "Granada", "Zaragoza", "Malaga", "Murcia", "Cordoba", "Alicante", "Valladolid", "Vigo", "Gijon", "Toledo", "Salamanca", "Pamplona", "Santander", "Logrono", "Burgos", "Almeria", "Huelva", "Leon", "Cadiz"},
	{"London", "Edinburgh", "Manchester", "Liverpool", "Bristol", "Cardiff", "Glasgow", "Birmingham", "Leeds", "Sheffield", "Newcastle", "Nottingham", "Leicester", "Coventry", "Norwich", "York", "Oxford", "Cambridge", "Southampton", "Portsmouth", "Plymouth", "Belfast", "Aberdeen", "Dundee"},
	{"Bruxelles", "Antwerpen", "Gent", "Liege", "Brugge", "Namur", "Leuven", "Mons", "Aalst", "Mechelen", "Charleroi", "Kortrijk", "Hasselt", "Oostende", "Tournai", "Genk", "Seraing", "Roeselare", "Verviers", "Mouscron", "La Louviere", "Sint-Niklaas", "Turnhout", "Arlon"},
}

type GameState struct {
	Tiles     [][]Tile      `json:"tiles"`
	Cities    []*City       `json:"cities"`
	Civs      []*Civ        `json:"civs"`
	Units     []*Unit       `json:"units"`
	Wonders   []WonderState `json:"wonders"`
	TechTree  []TechDef     `json:"techTree"`
	Turn      int           `json:"turn"`
	Phase     string        `json:"phase"`
	Events    []string      `json:"events"`
	Winner    string        `json:"winner"`
	Victory   string        `json:"victory"`
	Objective string        `json:"objective"`
}

// Globals

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

// Helpers

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
func cityHasWonder(cityID int) bool {
	for _, w := range state.Wonders {
		if w.CityID == cityID && w.CivID != -1 {
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

// Noise

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

// Map

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

// Yields

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

// Borders

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

// Settlement

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
	// Prefer safer expansion: avoid enemy armies, slightly prefer own military cover.
	for _, u := range state.Units {
		d := tileDist(x, y, u.X, u.Y)
		if d > 6 {
			continue
		}
		if u.CivID == civID {
			score += (6 - d) * 0.6
		} else {
			score -= (6 - d) * 1.6
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

func findBestSettlementFrom(civID, sx, sy int) (int, int, bool) {
	best, bx, by := -1.0, sx, sy
	found := false
	for i := 0; i < 220; i++ {
		x, y := rng.Intn(mapW), rng.Intn(mapH)
		s := settlementScore(x, y, civID)
		if s < 0 {
			continue
		}
		// Prefer closer viable spots so settlers don't wander forever.
		d := tileDist(x, y, sx, sy)
		s -= d * 0.8
		if s > best {
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
	if idx >= len(territoryCityNamePool[civID]) {
		idx = len(territoryCityNamePool[civID]) - 1
	}
	city := &City{ID: newID(), Name: territoryCityNamePool[civID][idx], CivID: civID, X: x, Y: y,
		Population: 1, FoodNeeded: 10, Buildings: []string{}, BorderRadius: 1, IsCoastal: isCoastalCity(x, y), Defense: 10, MaxDefense: 10}
	state.Tiles[y][x].CityID = city.ID
	state.Tiles[y][x].CivID = civID
	expandBorders(city)
	civ.Cities = append(civ.Cities, city.ID)
	state.Cities = append(state.Cities, city)
	return city
}

func addUnit(civID, x, y int, str int, kind string) bool {
	if x < 0 || x >= mapW || y < 0 || y >= mapH {
		return false
	}
	if !isLand(state.Tiles[y][x].terrain) {
		return false
	}
	for _, u := range state.Units {
		if u.X == x && u.Y == y {
			return false
		}
	}
	state.Units = append(state.Units, &Unit{
		ID:       newID(),
		CivID:    civID,
		X:        x,
		Y:        y,
		Strength: str,
		Type:     kind,
	})
	return true
}

func spawnUnitNearCity(city *City, str int, kind string) bool {
	if addUnit(city.CivID, city.X, city.Y, str, kind) {
		return true
	}
	for r := 1; r <= 2; r++ {
		for dy := -r; dy <= r; dy++ {
			for dx := -r; dx <= r; dx++ {
				if abs(dx) != r && abs(dy) != r {
					continue
				}
				x, y := city.X+dx, city.Y+dy
				if addUnit(city.CivID, x, y, str, kind) {
					return true
				}
			}
		}
	}
	return false
}

func spawnGarrisonForCity(city *City) bool {
	kind := "melee"
	civ := state.Civs[city.CivID]
	if civ.Era >= 2 && rng.Intn(3) == 0 {
		kind = "ranged"
	}
	if civ.Era >= 3 && civ.Strategy == "militarist" && rng.Intn(4) == 0 {
		kind = "cavalry"
	}
	return spawnUnitNearCity(city, 2+civ.Era, kind)
}

func spawnStartingSettlers() {
	for _, civ := range state.Civs {
		x, y, ok := findBestSettlement(civ.ID)
		if !ok {
			continue
		}
		addUnit(civ.ID, x, y, 1, "settler")
	}
}

func rebuildCivCityLists() {
	for _, civ := range state.Civs {
		civ.Cities = civ.Cities[:0]
	}
	for _, city := range state.Cities {
		if city.CivID >= 0 && city.CivID < len(state.Civs) {
			state.Civs[city.CivID].Cities = append(state.Civs[city.CivID].Cities, city.ID)
		}
	}
	for _, civ := range state.Civs {
		civ.Alive = len(civ.Cities) > 0
	}
}

func unitAt(x, y int) *Unit {
	for _, u := range state.Units {
		if u.X == x && u.Y == y {
			return u
		}
	}
	return nil
}

func nearestEnemyCity(u *Unit) *City {
	var best *City
	bestD := 1e9
	for _, c := range state.Cities {
		if c.CivID == u.CivID {
			continue
		}
		d := tileDist(u.X, u.Y, c.X, c.Y)
		if d < bestD {
			bestD = d
			best = c
		}
	}
	return best
}

func nearestEnemyUnit(u *Unit, maxDist float64) *Unit {
	var best *Unit
	bestD := 1e9
	for _, other := range state.Units {
		if other.CivID == u.CivID || other.Type == "settler" {
			continue
		}
		d := tileDist(u.X, u.Y, other.X, other.Y)
		if d <= maxDist && d < bestD {
			bestD = d
			best = other
		}
	}
	return best
}

func adjacentSupport(x, y, civID int, excludeID int) int {
	n := 0
	for _, u := range state.Units {
		if u.ID == excludeID || u.CivID != civID || u.Type == "settler" {
			continue
		}
		if abs(u.X-x) <= 1 && abs(u.Y-y) <= 1 {
			n++
		}
	}
	return n
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func nearestCityOf(civID, x, y int) *City {
	var best *City
	bestD := 1e9
	for _, c := range state.Cities {
		if c.CivID != civID {
			continue
		}
		d := tileDist(x, y, c.X, c.Y)
		if d < bestD {
			bestD = d
			best = c
		}
	}
	return best
}

func sign(v int) int {
	if v < 0 {
		return -1
	}
	if v > 0 {
		return 1
	}
	return 0
}

func resolveBattle(att, def *Unit) *Unit {
	typeBonus := func(kind string) int {
		switch kind {
		case "ranged":
			return 1
		case "cavalry":
			return 2
		default:
			return 0
		}
	}
	attRoll := att.Strength + typeBonus(att.Type) + rng.Intn(3)
	defRoll := def.Strength + typeBonus(def.Type) + rng.Intn(3)
	if attRoll >= defRoll {
		att.Strength = imax(1, att.Strength-1)
		return att
	}
	def.Strength = imax(1, def.Strength-1)
	return def
}

func removeUnitByID(id int) {
	for i, u := range state.Units {
		if u.ID == id {
			state.Units = append(state.Units[:i], state.Units[i+1:]...)
			return
		}
	}
}

func captureCity(city *City, newCivID int) {
	old := city.CivID
	city.CivID = newCivID
	city.BorderRadius = imax(1, city.BorderRadius-1)
	city.Defense = city.MaxDefense / 2
	state.Tiles[city.Y][city.X].CivID = newCivID
	state.Tiles[city.Y][city.X].CityID = city.ID
	for i := range state.Wonders {
		if state.Wonders[i].CityID == city.ID {
			state.Wonders[i].CivID = newCivID
		}
	}
	expandBorders(city)
	rebuildCivCityLists()
	state.Events = append(state.Events, fmt.Sprintf("%s capture %s", state.Civs[newCivID].Name, city.Name))
	if old >= 0 && old < len(state.Civs) && !state.Civs[old].Alive {
		state.Events = append(state.Events, fmt.Sprintf("%s est eliminee", state.Civs[old].Name))
	}
}

func siegeCity(attacker *Unit, city *City) bool {
	if attacker == nil || city == nil || city.CivID == attacker.CivID {
		return true
	}
	support := adjacentSupport(city.X, city.Y, attacker.CivID, attacker.ID)
	damage := imax(1, attacker.Strength/2+support)
	if attacker.Type == "ranged" {
		damage++
	}
	city.Defense -= damage
	state.Events = append(state.Events, fmt.Sprintf("%s assiege %s (%d/%d)", state.Civs[attacker.CivID].Name, city.Name, imax(0, city.Defense), city.MaxDefense))

	ret := 1
	if hasBuilding(city, "walls") {
		ret++
	}
	if city.MaxDefense >= 16 {
		ret++
	}
	attacker.Strength -= ret
	if attacker.Strength <= 0 {
		state.Events = append(state.Events, fmt.Sprintf("%s repousse l'assaut contre %s", state.Civs[city.CivID].Name, city.Name))
		removeUnitByID(attacker.ID)
		return false
	}
	if city.Defense <= 0 {
		captureCity(city, attacker.CivID)
	}
	return true
}

func cityUnderSiege(city *City) bool {
	for _, u := range state.Units {
		if u.Type == "settler" || u.CivID == city.CivID {
			continue
		}
		if abs(u.X-city.X) <= 1 && abs(u.Y-city.Y) <= 1 {
			return true
		}
	}
	return false
}

func runSettlers() {
	if len(state.Units) == 0 {
		return
	}
	occupied := make(map[[2]int]*Unit, len(state.Units))
	for _, ou := range state.Units {
		occupied[[2]int{ou.X, ou.Y}] = ou
	}
	consume := []int{}
	for _, u := range state.Units {
		if u.Type != "settler" {
			continue
		}
		civ := state.Civs[u.CivID]
		if !civ.Alive && len(civ.Cities) > 0 {
			continue
		}
		// Found immediately if location is valid.
		if settlementScore(u.X, u.Y, u.CivID) >= 0 {
			city := foundCity(u.CivID, u.X, u.Y)
			state.Events = append(state.Events, civ.Name+" fonde "+city.Name)
			spawnGarrisonForCity(city)
			consume = append(consume, u.ID)
			continue
		}

		tx, ty, ok := findBestSettlementFrom(u.CivID, u.X, u.Y)
		if !ok {
			continue
		}
		dx := sign(tx - u.X)
		dy := sign(ty - u.Y)
		candidates := [][2]int{
			{u.X + dx, u.Y + dy},
			{u.X + dx, u.Y},
			{u.X, u.Y + dy},
			{u.X - dx, u.Y},
			{u.X, u.Y - dy},
		}
		for _, c := range candidates {
			nx, ny := c[0], c[1]
			if nx < 0 || nx >= mapW || ny < 0 || ny >= mapH {
				continue
			}
			if !isLand(state.Tiles[ny][nx].terrain) {
				continue
			}
			if occupied[[2]int{nx, ny}] != nil {
				continue
			}
			delete(occupied, [2]int{u.X, u.Y})
			u.X, u.Y = nx, ny
			occupied[[2]int{u.X, u.Y}] = u
			break
		}
	}
	for _, id := range consume {
		// Cleanup occupancy before unit removal.
		for _, u := range state.Units {
			if u.ID == id {
				delete(occupied, [2]int{u.X, u.Y})
				break
			}
		}
		removeUnitByID(id)
	}
}

func runUnitsAndWars() {
	if len(state.Units) == 0 {
		return
	}
	occupied := make(map[[2]int]*Unit, len(state.Units))
	for _, ou := range state.Units {
		occupied[[2]int{ou.X, ou.Y}] = ou
	}

	// Basic production of new military units.
	for _, civ := range state.Civs {
		if !civ.Alive || len(civ.Cities) == 0 {
			continue
		}
		unitCount := 0
		for _, u := range state.Units {
			if u.CivID == civ.ID && u.Type != "settler" {
				unitCount++
			}
		}
		capacity := len(civ.Cities)*3 + 2
		if unitCount >= capacity || civ.Gold < 18 {
			continue
		}
		home := state.Cities[rng.Intn(len(state.Cities))]
		for home.CivID != civ.ID {
			home = state.Cities[rng.Intn(len(state.Cities))]
		}
		if spawnGarrisonForCity(home) {
			civ.Gold -= 18
			state.Events = append(state.Events, fmt.Sprintf("%s leve une unite a %s", civ.Name, home.Name))
		}
	}
	occupied = make(map[[2]int]*Unit, len(state.Units))
	for _, ou := range state.Units {
		occupied[[2]int{ou.X, ou.Y}] = ou
	}

	// Move units and resolve encounters.
	for _, u := range state.Units {
		if u.Type == "settler" {
			continue
		}
		civ := state.Civs[u.CivID]
		if !civ.Alive {
			continue
		}
		// Prefer nearby enemy armies first; otherwise push toward cities.
		targetUnit := nearestEnemyUnit(u, 7)
		target := nearestEnemyCity(u)
		tx, ty := u.X, u.Y
		if targetUnit != nil {
			tx, ty = targetUnit.X, targetUnit.Y
		} else if target != nil {
			tx, ty = target.X, target.Y
		} else {
			continue
		}
		if targetUnit == nil && target != nil && abs(u.X-target.X) <= 1 && abs(u.Y-target.Y) <= 1 {
			alive := siegeCity(u, target)
			if !alive {
				delete(occupied, [2]int{u.X, u.Y})
			}
			continue
		}
		dx := sign(tx - u.X)
		dy := sign(ty - u.Y)
		candidates := [][2]int{
			{u.X + dx, u.Y + dy},
			{u.X + dx, u.Y},
			{u.X, u.Y + dy},
			{u.X + dx, u.Y - dy},
			{u.X - dx, u.Y + dy},
		}

		nx, ny := u.X, u.Y
		for _, c := range candidates {
			tx, ty := c[0], c[1]
			if tx < 0 || tx >= mapW || ty < 0 || ty >= mapH {
				continue
			}
			if !isLand(state.Tiles[ty][tx].terrain) {
				continue
			}
			occ := occupied[[2]int{tx, ty}]
			if occ != nil && occ.CivID == u.CivID {
				continue
			}
			nx, ny = tx, ty
			break
		}

		if nx == u.X && ny == u.Y {
			continue
		}

		if enemy := occupied[[2]int{nx, ny}]; enemy != nil && enemy.CivID != u.CivID {
			winner := resolveBattle(u, enemy)
			loser := enemy
			if winner.ID != enemy.ID {
				loser = u
			}
			delete(occupied, [2]int{loser.X, loser.Y})
			removeUnitByID(loser.ID)
			if winner.ID == u.ID {
				delete(occupied, [2]int{u.X, u.Y})
				u.X, u.Y = nx, ny
				occupied[[2]int{u.X, u.Y}] = u
			}
			state.Events = append(state.Events, fmt.Sprintf("%s remporte un combat", state.Civs[winner.CivID].Name))
			continue
		}

		delete(occupied, [2]int{u.X, u.Y})
		u.X, u.Y = nx, ny
		occupied[[2]int{u.X, u.Y}] = u

		// City siege/capture.
		for _, c := range state.Cities {
			if c.X == u.X && c.Y == u.Y && c.CivID != u.CivID {
				alive := siegeCity(u, c)
				if !alive {
					delete(occupied, [2]int{u.X, u.Y})
				}
				break
			}
		}
	}
}

func runBorderPressure() {
	for _, city := range state.Cities {
		owner := city.CivID
		if owner < 0 || owner >= len(state.Civs) {
			continue
		}
		pressure := 1 + state.Civs[owner].Era/2 + city.Population/4
		if hasBuilding(city, "monument") {
			pressure++
		}
		if hasBuilding(city, "temple") {
			pressure++
		}
		r := city.BorderRadius + 1
		bestScore := -9999.0
		bestX, bestY := -1, -1
		for dy := -r; dy <= r; dy++ {
			for dx := -r; dx <= r; dx++ {
				nx, ny := city.X+dx, city.Y+dy
				if nx < 0 || nx >= mapW || ny < 0 || ny >= mapH {
					continue
				}
				if dx == 0 && dy == 0 {
					continue
				}
				if !isLand(state.Tiles[ny][nx].terrain) {
					continue
				}
				t := &state.Tiles[ny][nx]
				if t.CityID != -1 || t.CivID == owner {
					continue
				}
				// Don't instantly steal deeply-held cores.
				if t.CivID != -1 {
					if other := nearestCityOf(t.CivID, nx, ny); other != nil {
						if tileDist(nx, ny, other.X, other.Y) <= 2.2 {
							continue
						}
					}
				}
				d := tileDist(nx, ny, city.X, city.Y)
				score := float64(pressure) - d*1.2
				if t.CivID == -1 {
					score += 1.2
				}
				if score > bestScore {
					bestScore = score
					bestX, bestY = nx, ny
				}
			}
		}
		if bestX != -1 && bestScore > 0 {
			state.Tiles[bestY][bestX].CivID = owner
		}
	}
}

func checkVictory() {
	if state.Phase == "ended" {
		return
	}
	allObjectives := state.Objective == "" || state.Objective == "normal"

	alive := []*Civ{}
	for _, civ := range state.Civs {
		if civ.Alive {
			alive = append(alive, civ)
		}
	}
	if (allObjectives || state.Objective == "domination") && len(alive) == 1 {
		state.Phase = "ended"
		state.Winner = alive[0].Name
		state.Victory = "Domination"
		state.Events = append(state.Events, alive[0].Name+" remporte une victoire de Domination")
		return
	}

	if allObjectives || state.Objective == "science" {
		for _, civ := range state.Civs {
			if len(civ.KnownTechs) >= 18 {
				state.Phase = "ended"
				state.Winner = civ.Name
				state.Victory = "Science"
				state.Events = append(state.Events, civ.Name+" remporte une victoire Scientifique")
				return
			}
		}
	}

	if allObjectives || state.Objective == "culture" {
		wCount := map[int]int{}
		for _, w := range state.Wonders {
			if w.CivID != -1 {
				wCount[w.CivID]++
			}
		}
		for civID, n := range wCount {
			if n >= 3 {
				state.Phase = "ended"
				state.Winner = state.Civs[civID].Name
				state.Victory = "Culture"
				state.Events = append(state.Events, state.Civs[civID].Name+" remporte une victoire Culturelle")
				return
			}
		}
	}

	if (allObjectives || state.Objective == "time") && state.Turn >= maxTurns {
		best := state.Civs[0]
		bestScore := -1
		for _, civ := range state.Civs {
			pop := 0
			for _, city := range state.Cities {
				if city.CivID == civ.ID {
					pop += city.Population
				}
			}
			score := civ.Science + civ.Gold + pop*10 + len(civ.Cities)*35 + civ.Era*60
			if score > bestScore {
				bestScore = score
				best = civ
			}
		}
		state.Phase = "ended"
		state.Winner = best.Name
		state.Victory = "Temps"
		state.Events = append(state.Events, "Fin du temps: "+best.Name+" gagne aux points")
	}
}

func updateVisibility() {
	// Reset current visibility every turn; exploration stays once discovered.
	for y := 0; y < mapH; y++ {
		for x := 0; x < mapW; x++ {
			state.Tiles[y][x].Visible = false
		}
	}

	// Before first settlements, show the whole map instead of a fully black screen.
	if len(state.Cities) == 0 {
		for y := 0; y < mapH; y++ {
			for x := 0; x < mapW; x++ {
				state.Tiles[y][x].Visible = true
				state.Tiles[y][x].Explored = true
			}
		}
		return
	}

	// Aggregate FoW from all cities (shared "observer" view for the simulation).
	for _, city := range state.Cities {
		vr := city.BorderRadius + 1
		for dy := -vr; dy <= vr; dy++ {
			for dx := -vr; dx <= vr; dx++ {
				if dx*dx+dy*dy > vr*vr {
					continue
				}
				nx, ny := city.X+dx, city.Y+dy
				if nx < 0 || nx >= mapW || ny < 0 || ny >= mapH {
					continue
				}
				state.Tiles[ny][nx].Visible = true
				state.Tiles[ny][nx].Explored = true
			}
		}
	}
	for _, u := range state.Units {
		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				nx, ny := u.X+dx, u.Y+dy
				if nx < 0 || nx >= mapW || ny < 0 || ny >= mapH {
					continue
				}
				state.Tiles[ny][nx].Visible = true
				state.Tiles[ny][nx].Explored = true
			}
		}
	}
}

// Tech AI

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
		state.Events = append(state.Events, civ.Name+" recoit une technologie gratuite: "+techByID[best].Name)
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

func effectiveTechCost(civ *Civ, td *TechDef) int {
	if td == nil {
		return 999999
	}
	known := len(civ.KnownTechs)
	// Progressive scaling slows late-game beelining.
	scale := 1.0 + float64(known)*0.09 + float64(td.Era)*0.12
	if scale < 1 {
		scale = 1
	}
	return int(math.Round(float64(td.Cost) * scale))
}

// Build AI

func aiChooseBuild(city *City) string {
	civ := state.Civs[city.CivID]
	if !cityHasWonder(city.ID) {
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
	}
	for _, bID := range strategyBuildOrder[civ.Strategy] {
		if bID == "settler" {
			if city.Population < 2 {
				continue
			}
			settlers := 0
			for _, u := range state.Units {
				if u.CivID == civ.ID && u.Type == "settler" {
					settlers++
				}
			}
			if settlers >= 2 || len(civ.Cities) >= 2+state.Turn/14 {
				continue
			}
			return "settler"
		}
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
	for _, bID := range strategyBuildOrder[civ.Strategy] {
		if bID == "settler" {
			continue
		}
		if def, ok := buildings[bID]; ok {
			if def.RequiresTech != "" && !hasTech(civ, def.RequiresTech) {
				continue
			}
			if bID == "harbor" && !city.IsCoastal {
				continue
			}
			return bID
		}
	}
	return "monument"
}

// Turn

func processTurn() {
	stateMu.Lock()
	defer stateMu.Unlock()
	if state.Phase != "running" {
		return
	}
	state.Turn++
	if len(state.Events) > 16 {
		state.Events = state.Events[len(state.Events)-16:]
	}

	if state.Turn == 1 {
		state.Events = append(state.Events, "Les colons partent fonder leurs capitales...")
		updateVisibility()
	}

	// Reset per-turn yields before recomputing economy.
	for _, civ := range state.Civs {
		if !civ.Alive {
			continue
		}
		civ.Science = 0
		civ.Culture = 0
	}

	// Cities
	for _, city := range state.Cities {
		city.MaxDefense = 10 + state.Civs[city.CivID].Era*2
		if hasBuilding(city, "walls") {
			city.MaxDefense += 4
		}
		if city.Defense < city.MaxDefense && !cityUnderSiege(city) {
			city.Defense++
		}
		food, prod, gold, sci, cult := calcYields(city)
		city.YieldFood, city.YieldProd, city.YieldGold, city.YieldScience, city.YieldCulture = food, prod, gold, sci, cult
		civ := state.Civs[city.CivID]
		civ.Gold += gold
		civ.Science += sci + imax(1, city.Population/4)
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
			expandBorders(city)
			state.Events = append(state.Events, city.Name+": frontieres etendues")
		}
		if city.CurrentBuild == "" {
			city.CurrentBuild = aiChooseBuild(city)
		}
		city.BuildProgress += prod
		if len(city.CurrentBuild) > 7 && city.CurrentBuild[:7] == "wonder:" {
			wID := city.CurrentBuild[7:]
			wd := wonderDefs[wID]
			if wonderBuilt(wID) || cityHasWonder(city.ID) {
				city.BuildProgress = 0
				city.CurrentBuild = ""
				continue
			}
			if city.BuildProgress >= wd.Cost {
				for i := range state.Wonders {
					if state.Wonders[i].ID == wID {
						state.Wonders[i].CivID = city.CivID
						state.Wonders[i].CityID = city.ID
						state.Wonders[i].Turn = state.Turn
						break
					}
				}
				state.Events = append(state.Events, fmt.Sprintf("%s construit %s !", civ.Name, wd.Name))
				if wd.FreeTech {
					grantFreeTech(civ)
				}
				spawnGarrisonForCity(city)
				city.BuildProgress = 0
				city.CurrentBuild = ""
			}
		} else if city.CurrentBuild == "settler" {
			if city.BuildProgress >= settlerCost {
				addUnit(city.CivID, city.X, city.Y, 1, "settler")
				if city.Population > 1 {
					city.Population--
					city.FoodNeeded = city.Population * 10
					if city.FoodBin > city.FoodNeeded {
						city.FoodBin = city.FoodNeeded
					}
				}
				state.Events = append(state.Events, fmt.Sprintf("%s forme un colon", city.Name))
				city.BuildProgress = 0
				city.CurrentBuild = ""
			}
		} else if def, ok := buildings[city.CurrentBuild]; ok && city.BuildProgress >= def.Cost {
			city.Buildings = append(city.Buildings, city.CurrentBuild)
			state.Events = append(state.Events, fmt.Sprintf("%s construit %s", city.Name, def.Name))
			if city.CurrentBuild == "barracks" || city.CurrentBuild == "forge" {
				spawnGarrisonForCity(city)
			}
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

	// Research (after current-turn yields are computed).
	for _, civ := range state.Civs {
		if !civ.Alive {
			continue
		}
		if civ.CurrentResearch == "" {
			civ.CurrentResearch = aiChooseResearch(civ)
		}
		if civ.CurrentResearch == "" {
			civ.ResearchProg = 0
			continue
		}
		td := techByID[civ.CurrentResearch]
		if td == nil {
			civ.CurrentResearch = ""
			civ.ResearchProg = 0
			continue
		}
		cost := effectiveTechCost(civ, td)
		civ.ScienceBin += civ.Science
		civ.ResearchProg = int(math.Min(100, math.Round((float64(civ.ScienceBin)/float64(cost))*100)))
		if civ.ScienceBin >= cost {
			civ.ScienceBin -= cost
			civ.KnownTechs = append(civ.KnownTechs, td.ID)
			civ.Era = civEra(civ)
			state.Events = append(state.Events, fmt.Sprintf("%s decouvre %s", civ.Name, td.Name))
			civ.CurrentResearch = aiChooseResearch(civ)
			civ.ResearchProg = 0
		}
	}

	runSettlers()
	runBorderPressure()
	runUnitsAndWars()
	checkVictory()
	updateVisibility()
}

// Init

func validObjective(objective string) string {
	switch objective {
	case "domination", "science", "culture", "time":
		return objective
	default:
		return "normal"
	}
}

func objectiveLabel(objective string) string {
	switch objective {
	case "domination":
		return "Domination"
	case "science":
		return "Science"
	case "culture":
		return "Culture"
	case "time":
		return "Temps"
	default:
		return "Normal"
	}
}

func initGame(objective string, running bool) {
	nextID = 0
	initTechIndex()
	objective = validObjective(objective)
	civs := make([]*Civ, numCivs)
	for i, d := range territoryData {
		civs[i] = &Civ{ID: i, Name: d.name, Color: d.color, Strategy: d.strategy, Gold: 50, Cities: []int{}, Alive: true, KnownTechs: []string{}}
	}
	wonders := []WonderState{}
	for _, wd := range wonderDefs {
		wonders = append(wonders, WonderState{ID: wd.ID, Name: wd.Name, CivID: -1, CityID: -1})
	}
	phase := "setup"
	events := []string{"Choisis un mode pour lancer la simulation."}
	if running {
		phase = "running"
		events = []string{"Objectif: " + objectiveLabel(objective), "Les civilisations s'eveillent..."}
	}
	state = GameState{Tiles: generateMap(), Cities: []*City{}, Civs: civs, Units: []*Unit{}, Wonders: wonders, TechTree: techTree, Turn: 0, Phase: phase, Events: events, Objective: objective}
	spawnStartingSettlers()
	updateVisibility()
}

// Broadcast & HTTP

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
		text := string(msg)
		if strings.HasPrefix(text, "start:") {
			stateMu.Lock()
			initGame(strings.TrimPrefix(text, "start:"), true)
			stateMu.Unlock()
			broadcast()
		} else if text == "restart" {
			stateMu.Lock()
			initGame(state.Objective, true)
			stateMu.Unlock()
			broadcast()
		} else if text == "menu" {
			stateMu.Lock()
			initGame("normal", false)
			stateMu.Unlock()
			broadcast()
		}
	}
}

func main() {
	initGame("normal", false)
	go gameLoop()
	http.Handle("/", http.FileServer(http.Dir("./static")))
	http.HandleFunc("/ws", wsHandler)
	log.Println("Civ sim Phase 2 - http://localhost:8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
