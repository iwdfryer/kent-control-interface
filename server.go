package main

import (
	"flag"
	"log"
	"net/http"
)

func main() {

	listen := flag.String("listen", ":8080", "browser server listen address")
	dir := flag.String("dir", ".", "browser server directory to serve")
	flag.Parse()

	//browser server
	log.Printf("listening on %q...", *listen)
	log.Fatal(http.ListenAndServe(*listen, http.FileServer(http.Dir(*dir))))
}
