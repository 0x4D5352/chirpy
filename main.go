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
	"time"

	"github.com/0x4D5352/chirpy/internal/database"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
	godotenv.Load()

	log.Println("Setting up Database...")
	dbURL := os.Getenv("DB_URL")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal(err)
	}
	dbQueries := database.New(db)

	log.Println("Setting up Server...")
	apiCfg := apiConfig{
		db:       dbQueries,
		platform: os.Getenv("PLATFORM"),
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

	log.Println("Setting up admin endpoints...")
	mux.HandleFunc("GET /admin/metrics", apiCfg.checkMetrics)
	mux.HandleFunc("POST /admin/reset", apiCfg.resetMetrics)

	log.Println("Setting up user creation endpoint...")
	mux.HandleFunc("POST /api/users", apiCfg.createUser)

	log.Println("Setting up chrips endpoint...")
	mux.HandleFunc("POST /api/chirps", apiCfg.postChirp)

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
	db             *database.Queries
	platform       string
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
	if cfg.platform != "dev" {
		log.Println("Unauthorized request to reset endpoint!")
		log.Printf("%+v", req)
		w.WriteHeader(http.StatusForbidden)
		return
	}

	log.Println("Resetting metrics...")
	cfg.fileServerHits.Store(0)

	log.Println("Resetting database...")
	err := cfg.db.ResetUsers(req.Context())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Fatal(err)
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	int, err := io.WriteString(w, "Metrics reset!")
	if err != nil {
		log.Fatal(err, int)
	}
}

type Post struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Body      string    `json:"body"`
	UserID    uuid.UUID `json:"user_id"`
}

func (cfg *apiConfig) postChirp(w http.ResponseWriter, req *http.Request) {
	log.Println("Chirp received!")
	type reqBody struct {
		Body   string    `json:"body"`
		UserID uuid.UUID `json:"user_id"`
	}
	decoder := json.NewDecoder(req.Body)
	rb := reqBody{}
	err := decoder.Decode(&rb)
	if err != nil {
		log.Printf("Error decoding body: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
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
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write(resp)
		return
	}

	contents := strings.Fields(rb.Body)
	for i, word := range contents {
		lw := strings.ToLower(word)
		if lw != "kerfuffle" && lw != "sharbert" && lw != "fornax" {
			continue
		}
		contents[i] = "****"
	}

	chirp, err := cfg.db.CreateChirp(req.Context(), database.CreateChirpParams{
		Body:   strings.Join(contents, " "),
		UserID: rb.UserID,
	})
	if err != nil {
		log.Printf("Error creating Chirp: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	resp, err := json.Marshal(Post{
		ID:        chirp.ID,
		CreatedAt: chirp.CreatedAt,
		UpdatedAt: chirp.UpdatedAt,
		Body:      chirp.Body,
		UserID:    chirp.UserID,
	})
	if err != nil {
		log.Printf("Error encoding response: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	w.Write(resp)
	log.Println("Chirp posted!")
}

type User struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email     string    `json:"email"`
}

func (cfg *apiConfig) createUser(w http.ResponseWriter, req *http.Request) {
	log.Println("User creation requested!")
	type reqBody struct {
		Email string `json:"email"`
	}
	// TODO: add validation?
	decoder := json.NewDecoder(req.Body)
	rb := reqBody{}
	err := decoder.Decode(&rb)
	if err != nil {
		log.Printf("Error decoding body: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	user, err := cfg.db.CreateUser(req.Context(), rb.Email)
	// TODO: send invalid response body
	if err != nil {
		log.Printf("Error creating user: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	resp, err := json.Marshal(User{
		ID:        user.ID,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
		Email:     user.Email,
	})
	if err != nil {
		log.Printf("Error encording response: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	w.Write(resp)
	log.Println("User created!")
}
