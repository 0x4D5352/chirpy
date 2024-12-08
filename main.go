package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync/atomic"

	"github.com/0x4D5352/chirpy/internal/database"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
	log.Println("Setting up Database...")
	godotenv.Load()
	dbURL := os.Getenv("DB_URL")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal(err)
	}
	dbQueries := database.New(db)
	log.Println("Setting up Server...")
	apiCfg := apiConfig{
		queries: dbQueries,
	}
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

// TODO: Decide if you should be storing the server in the config or not.
type apiConfig struct {
	fileServerHits atomic.Int32
	queries        *database.Queries
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
