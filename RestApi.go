package main

import (
	"net/http"
	"log"
	"encoding/json"
	"github.com/gorilla/mux"
)

func SetupRouteApiV1(router *mux.Router) {
	router.HandleFunc("/discover-printers", discoverPrinters)
	router.HandleFunc("/printers", handleGetPrinters).Methods("GET")
}

func discoverPrinters(w http.ResponseWriter, r *http.Request) {
	log.Println("Discovering printers...")

	printers := UdevPrinterDiscovery()

	js, err := json.Marshal(printers)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

func handleGetPrinters(w http.ResponseWriter, r *http.Request) {
}

