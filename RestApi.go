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
	router.HandleFunc("/printers", handleAddPrinter).Methods("POST")
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

type RestPrinterSettings struct {
	Name string `json:"name"`
	DevicePath string `json:"device_path"`
	BaudRate uint `json:"baud_rate"`
	Default bool `json:"default"`
	Width uint `json:"width"`
	Height uint `json:"height"`
	Depth uint `json:"depth"`
	Stopped bool `json:"stopped"`
}

func handleGetPrinters(w http.ResponseWriter, r *http.Request) {
}

func printerSettingsFromRest(t RestPrinterSettings, p *PrinterSettings) {
	p.Name = t.Name
	p.DevicePath = t.DevicePath
	p.BaudRate = t.BaudRate
	p.PrintArea.Width = t.Width
	p.PrintArea.Height = t.Height
	p.PrintArea.Depth = t.Depth
	p.Stopped = t.Stopped
}

func printerSettingsToRest(t* RestPrinterSettings, p PrinterSettings) {
	t.Name = p.Name
	t.DevicePath = p.DevicePath
	t.BaudRate = p.BaudRate
	t.Width = p.PrintArea.Width
	t.Height = p.PrintArea.Height
	t.Depth = p.PrintArea.Depth
	t.Stopped = p.Stopped
	t.Default = true // TODO
}

func handleAddPrinter(w http.ResponseWriter, r *http.Request) {
	var t RestPrinterSettings
	t.BaudRate = 115200
	
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&t)
	
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	if t.Name == "" || t.DevicePath == "" || t.BaudRate == 0 {
		http.Error(w, "Bad printer parameters", http.StatusBadRequest)
		return
	}
	
	// Send HTTP Location
	var p PrinterSettings
	printerSettingsFromRest(t, &p)
	
	printerName := addPrinter(p)
	
	saveConfig()
	
	w.Header().Set("Location", "http://" + r.Header.Get("Host") + "/api/v1/printers/" + printerName)
	w.WriteHeader(http.StatusCreated)
}
