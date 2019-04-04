package main

//go:generate $GOPATH/bin/go-bindata -pkg $GOPACKAGE -o assets.go -prefix web/dist web/dist

import (
	"net/http"
	"log"
	"flag"
	"github.com/gorilla/websocket"
	"github.com/gorilla/mux"
	"github.com/spf13/viper"
)

var httpAddr = flag.String("address", ":8181", "HTTP service address")
var wsUpgrader = websocket.Upgrader{}

func main() {
	flag.Parse()

	loadConfig()

	router := mux.NewRouter()
	router.HandleFunc("/websocket", handleWebsocket)

	SetupRouteApiV1(router.PathPrefix("/api/v1").Subrouter())

	router.PathPrefix("/").HandlerFunc(serveStatic)

	err := http.ListenAndServe(*httpAddr, router);
	if err != nil {
			log.Fatal("HTTP error: ", err)
	}
}

func loadConfig() {
	var configuration Configuration

	viper.SetConfigName("dashprint")
	viper.AddConfigPath("$HOME/.local/share")

	if err := viper.ReadInConfig(); err != nil {
		log.Println("Cannot load config file: ", err)
		return
	}

	err := viper.Unmarshal(&configuration)
	if err != nil {
		log.Println("Unable to decode config file: ", err)
	}

	loadPrinters(configuration)
}

func handleWebsocket(w http.ResponseWriter, r *http.Request) {
	c, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("WS Upgrade: ", err)
		return
	}

	defer c.Close()
	for {
		mt, message, err := c.ReadMessage()

		if err != nil {
			log.Println("WS read: ", err)
			break
		}

		_ = mt;
		_ = message;
		// err = c.WriteMessage...
	}
}

func serveStatic(w http.ResponseWriter, r *http.Request) {
	log.Print("Request: ", *r)

	path := r.URL.Path[1:]
	if path == "" {
		path = "index.html";
	}

	data, err := Asset(path)
	if err != nil {
		log.Println("Cannot find asset ", path)
		http.Error(w, "File Not Found", http.StatusNotFound)
		return
	}

	contentType := http.DetectContentType(data)
	w.Header().Set("Content-Type", contentType)
	w.Write(data)
}

