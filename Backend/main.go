package main

import (
	"fmt"
	"net/http"
)

func main() {
	// Une route simple pour tester
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Bienvenue sur Tamagotchi Hive !")
	})

	fmt.Println("Le serveur démarre sur http://localhost:8080")

	// Lancement du serveur sur le port 8080
	if err := http.ListenAndServe(":8080", nil); err != nil {
		panic(err)
	}
}
