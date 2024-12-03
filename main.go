package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"sync/atomic"
)

func main() {
	fmt.Println("Setting up Server...")
	var apiCfg apiConfig
	mux := http.NewServeMux()
	serv := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}
	fmt.Println("Setting up web app...")
	handler := http.StripPrefix("/app/", http.FileServer(http.Dir(".")))
	mux.Handle("/app/", apiCfg.middlewareMetricsInc(handler))
	fmt.Println("Setting up health endpoint...")
	mux.HandleFunc("/healthz", checkHealth)
	fmt.Println("Setting up metrics endpoint...")
	mux.HandleFunc("/metrics", apiCfg.checkMetrics)
	mux.HandleFunc("/reset", apiCfg.resetMetrics)
	fmt.Println("Starting Server...")
	log.Fatal(serv.ListenAndServe())
}

func checkHealth(w http.ResponseWriter, req *http.Request) {
	fmt.Println("Health endpoint hit!")
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	int, err := io.WriteString(w, "OK")
	if err != nil {
		log.Fatal(err, int)
	}
}

type apiConfig struct {
	fileServerHits atomic.Int32
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	handler := func(w http.ResponseWriter, req *http.Request) {
		fmt.Println("Incrementing pagecount...")
		cfg.fileServerHits.Add(1)
		next.ServeHTTP(w, req)
	}
	return http.HandlerFunc(handler)
}

func (cfg *apiConfig) checkMetrics(w http.ResponseWriter, req *http.Request) {
	fmt.Println("Checking metrics...")
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	int, err := io.WriteString(w, fmt.Sprintf("Hits: %v", cfg.fileServerHits.Load()))
	if err != nil {
		log.Fatal(err, int)
	}
}

func (cfg *apiConfig) resetMetrics(w http.ResponseWriter, req *http.Request) {
	fmt.Println("Resetting metrics...")
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	cfg.fileServerHits.Store(0)
	int, err := io.WriteString(w, "Metrics reset!")
	if err != nil {
		log.Fatal(err, int)
	}
}
