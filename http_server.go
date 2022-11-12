//go:build !wasm

// file: http_server.go

package main

import (
	"log"
	"net/http"

	"github.com/NYTimes/gziphandler"
	"github.com/maxence-charriere/go-app/v9/pkg/app"
)

func initServer() {
	withGz := gziphandler.GzipHandler(&app.Handler{
		Name:        "cyber-witness",
		Description: "Cyber Witness - Liquid democracy politics simulator based on personal reputation index",
		Styles: []string{
			"https://assets.ubuntu.com/v1/vanilla-framework-version-3.8.0.min.css",
			"https://use.fontawesome.com/releases/v6.2.0/css/all.css",
		},
		Scripts: []string{},
	})
	http.Handle("/", withGz)

	if err := http.ListenAndServe(":7000", nil); err != nil {
		log.Fatal(err)
	}
}
