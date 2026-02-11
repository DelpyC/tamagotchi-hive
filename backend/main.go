package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"tama/backend/game"
)

func main() {
	myWorld := game.NewWorld(20, 10)
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
		err := json.NewEncoder(w).Encode(myWorld.Grid)
		if err != nil {
			http.Error(w, "failed to encode map", http.StatusInternalServerError)
		}
	})

	fmt.Println("Serveur lancé sur http://localhost:8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal(err)
	}
}
