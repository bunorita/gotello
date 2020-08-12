package controllers

import (
	"fmt"
	"net/http"
	"text/template"

	"github.com/bunorita/gotello/config"
)

func viewIndexHandler(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("app/views/index.html")
	if err == nil {
		err = t.Execute(w, nil)
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func StartWebServer() error {
	http.HandleFunc("/", viewIndexHandler)
	return http.ListenAndServe(
		fmt.Sprintf("%s:%d", config.Config.Address, config.Config.Port),
		nil,
	)
}