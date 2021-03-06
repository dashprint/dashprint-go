package main

//go:generate $GOPATH/bin/go-bindata -pkg $GOPACKAGE -o assets.go -prefix web/dist web/dist/...

import (
	"net/http"
	"log"
	"flag"
	"mime"
	"path"
	"github.com/gorilla/websocket"
	"github.com/gorilla/mux"
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
	defer r.Body.Close()
	log.Print("Request: ", *r)

	fpath := r.URL.Path[1:]
	if fpath == "" {
		fpath = "index.html";
	}

	data, err := Asset(fpath)
	if err != nil {
		log.Println("Cannot find asset ", fpath)
		http.Error(w, "File Not Found", http.StatusNotFound)
		return
	}

	ext := path.Ext(fpath)
	contentType := mime.TypeByExtension(ext)
	w.Header().Set("Content-Type", contentType)
	w.Write(data)
}

