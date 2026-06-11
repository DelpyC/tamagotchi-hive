package game

type CellDTO struct {
	Terrain string `json:"terrain"`
}

type EntityDTO struct {
	Type       string `json:"type"`
	X          int    `json:"x"`
	Y          int    `json:"y"`
	Name       string `json:"name"`
	Population int    `json:"population"`
	Color      string `json:"color"`
}

type WorldDTO struct {
	Width    int         `json:"width"`
	Height   int         `json:"height"`
	Cells    []CellDTO   `json:"cells"`
	Entities []EntityDTO `json:"entities"`
	Events   []string    `json:"events"`
}

func (w *World) ToDTO() WorldDTO {
	w.mu.RLock()
	defer w.mu.RUnlock()

	cells := make([]CellDTO, 0, w.Width*w.Height)
	for r := 0; r < w.Height; r++ {
		for q := 0; q < w.Width; q++ {
			hex, exists := w.Grid[serializePos(q, r)]
			if !exists {
				cells = append(cells, CellDTO{Terrain: "ocean"})
				continue
			}
			cells = append(cells, CellDTO{Terrain: hex.Terrain})
		}
	}

	return WorldDTO{
		Width:    w.Width,
		Height:   w.Height,
		Cells:    cells,
		Entities: []EntityDTO{},
		Events:   []string{},
	}
}
