package main

import (
	"net/http"
	"log"
	"flag"
	"fmt"
	"encoding/json"
)

var httpAddr = flag.String("address", ":8181", "HTTP service address")

func main() {
	flag.Parse()

	http.HandleFunc("/", serveStatic)
	http.HandleFunc("/api/v1/discover-printers", discoverPrinters)

	err := http.ListenAndServe(*httpAddr, nil);
	if err != nil {
			log.Fatal("HTTP error: ", err)
	}
}

func discoverPrinters(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Discovering printers...")

	printers := UdevPrinterDiscovery()

	js, err := json.Marshal(printers)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

func serveStatic(w http.ResponseWriter, r *http.Request) {
	log.Print("Request: ", *r)
}

