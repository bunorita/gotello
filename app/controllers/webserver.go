package controllers

import (
	"fmt"
	"net/http"
	"text/template"

	"github.com/bunorita/gotello/config"
)

func getTemplate(temp string) (*template.Template, error) {
	return template.ParseFiles("app/views/layout.html", temp)
}

func viewIndexHandler(w http.ResponseWriter, r *http.Request) {
	t, err := getTemplate("app/views/index.html")
	if err == nil {
		err = t.Execute(w, nil)
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func StartWebServer() error {
	http.HandleFunc("/", viewIndexHandler)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	return http.ListenAndServe(
		fmt.Sprintf("%s:%d", config.Config.Address, config.Config.Port),
		nil,
	)
}
