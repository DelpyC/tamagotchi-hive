package controller

import (
	"html/template"
	"net/http"
)

func renderTemplate(w http.ResponseWriter, filename string, data map[string]string) {
	tmpl := template.Must(template.ParseFiles("template/" + filename))
	err := tmpl.Execute(w, data)
	if err != nil {
		return
	}
}
