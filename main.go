package main

import (
	"log"
	"net/http"
)

func main() {
	mux := http.NewServeMux()
	serv := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}
	mux.Handle("/", http.FileServer(http.Dir(".")))
	log.Fatal(serv.ListenAndServe())
}
