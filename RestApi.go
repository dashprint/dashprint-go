package main

import (
	"net/http"
	"log"
	"encoding/json"
	"github.com/gorilla/mux"
)

func SetupRouteApiV1(router *mux.Router) {
	router.HandleFunc("/printers/discover", discoverPrinters)
	router.HandleFunc("/printers", handleGetPrinters).Methods("GET")
	router.HandleFunc("/printers", handleAddPrinter).Methods("POST")

	router.HandleFunc("/printers/{printerId}", handleGetPrinter).Methods("GET")
	router.HandleFunc("/printers/{printerId}", handleSetupPrinter).Methods("PUT")

	router.HandleFunc("/printers/{printerId}/job", handleSubmitJob).Methods("POST")
	router.HandleFunc("/printers/{printerId}/job", handleModifyJob).Methods("PUT")
	router.HandleFunc("/printers/{printerId}/job", handleGetJob).Methods("GET")

	router.HandleFunc("/printers/{printerId}/temperatures", handleGetPrinterTemperatures).Methods("GET")
	router.HandleFunc("/printers/{printerId}/temperatures", handleSetPrinterTemperatures).Methods("SET")

	router.HandleFunc("/files", handleListFiles).Methods("GET")
	router.HandleFunc("/files/{file}", handleDownloadFile).Methods("GET")
	router.HandleFunc("/files/{file}", handleUploadFile).Methods("PUT")
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
	printerMutex.RLock()
	defer printerMutex.RUnlock()

	jsonData := make(map[string]RestPrinterSettings)

	for uniqueName, printer := range printers {
		var ps RestPrinterSettings
		printerSettingsToRest(&ps, printer.PrinterSettings)
		jsonData[uniqueName] = ps
	}

	js, err := json.Marshal(jsonData)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

func handleGetPrinter(w http.ResponseWriter, r *http.Request) {
	printerMutex.RLock()
	defer printerMutex.RUnlock()

	vars := mux.Vars(r)

	if printer, ok := printers[vars["printerId"]]; ok {
		var rps RestPrinterSettings
		printerSettingsToRest(&rps, printer.PrinterSettings)

		js, err := json.Marshal(rps)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(js)
	} else {
		http.NotFound(w, r)
	}
}

func handleSetupPrinter(w http.ResponseWriter, r *http.Request) {
	// TODO
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
	t.Default = defaultPrinter == p.UniqueName
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

func handleSubmitJob(w http.ResponseWriter, r *http.Request) {
	// TODO
}

func handleModifyJob(w http.ResponseWriter, r *http.Request) {
	// TODO
}

func handleGetJob(w http.ResponseWriter, r *http.Request) {
	// TODO
}

func handleGetPrinterTemperatures(w http.ResponseWriter, r *http.Request) {
	// TODO
}

func handleSetPrinterTemperatures(w http.ResponseWriter, r *http.Request) {
	// TODO
}

func handleListFiles(w http.ResponseWriter, r *http.Request) {
	// TODO
}

func handleDownloadFile(w http.ResponseWriter, r *http.Request) {
	// TODO
}

func handleUploadFile(w http.ResponseWriter, r *http.Request) {
	// TODO
}

