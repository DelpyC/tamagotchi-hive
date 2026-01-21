package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"tama/backend/game"
)

func main() {
	myWorld := game.NewWorld(20, 10)

	http.HandleFunc("/api/map", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(myWorld.Grid)
	})

	fmt.Println("Serveur lancé sur http://localhost:8080")
	http.ListenAndServe(":8080", nil)
}
