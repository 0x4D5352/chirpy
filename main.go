package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync/atomic"
)

func main() {
	log.Println("Setting up Server...")
	var apiCfg apiConfig
	mux := http.NewServeMux()
	serv := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}
	log.Println("Setting up web app...")
	handler := http.StripPrefix("/app/", http.FileServer(http.Dir(".")))
	mux.Handle("/app/", apiCfg.middlewareMetricsInc(handler))
	log.Println("Setting up health endpoint...")
	mux.HandleFunc("GET /api/healthz", checkHealth)
	log.Println("Setting up metrics endpoint...")
	mux.HandleFunc("GET /admin/metrics", apiCfg.checkMetrics)
	mux.HandleFunc("POST /admin/reset", apiCfg.resetMetrics)
	log.Println("Setting up validation endpoint...")
	mux.HandleFunc("POST /api/validate_chirp", validateChirp)
	log.Println("Starting Server...")
	log.Fatal(serv.ListenAndServe())
}

func checkHealth(w http.ResponseWriter, req *http.Request) {
	log.Println("Health endpoint hit!")
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
		log.Println("Incrementing pagecount...")
		cfg.fileServerHits.Add(1)
		next.ServeHTTP(w, req)
	}
	return http.HandlerFunc(handler)
}

func (cfg *apiConfig) checkMetrics(w http.ResponseWriter, req *http.Request) {
	log.Println("Checking metrics...")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	body := fmt.Sprintf(`<html>
  <body>
    <h1>Welcome, Chirpy Admin</h1>
    <p>Chirpy has been visited %d times!</p>
  </body>
</html>`, cfg.fileServerHits.Load())
	int, err := io.WriteString(w, body)
	if err != nil {
		log.Fatal(err, int)
	}
}

func (cfg *apiConfig) resetMetrics(w http.ResponseWriter, req *http.Request) {
	log.Println("Resetting metrics...")
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	cfg.fileServerHits.Store(0)
	int, err := io.WriteString(w, "Metrics reset!")
	if err != nil {
		log.Fatal(err, int)
	}
}

func validateChirp(w http.ResponseWriter, req *http.Request) {
	log.Println("Chirp sent in for validation!")
	type reqBody struct {
		Body string `json:"body"`
	}
	decoder := json.NewDecoder(req.Body)
	rb := reqBody{}
	err := decoder.Decode(&rb)
	if err != nil {
		log.Printf("Error decoding body: %s", err)
		w.WriteHeader(500)
		return
	}
	type invalidResponse struct {
		Error string `json:"error"`
	}
	log.Println(rb.Body)
	if len(rb.Body) > 140 {
		resp, err := json.Marshal(invalidResponse{Error: "Chirp is too long"})
		log.Println("too long!")
		if err != nil {
			log.Printf("Error encording error: %s", err)
			w.WriteHeader(500)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(400)
		w.Write(resp)
		return
	}
	type validResponse struct {
		CleanedBody string `json:"cleaned_body"`
	}
	contents := strings.Fields(rb.Body)
	for i, word := range contents {
		lw := strings.ToLower(word)
		if lw != "kerfuffle" && lw != "sharbert" && lw != "fornax" {
			continue
		}
		contents[i] = "****"
	}
	resp, err := json.Marshal(validResponse{
		CleanedBody: strings.Join(contents, " "),
	})
	if err != nil {
		log.Printf("Error encording error: %s", err)
		w.WriteHeader(500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	w.Write(resp)
	log.Println("chirp validated!")
}
