package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync/atomic"
	"time"

	"github.com/google/uuid"

	"github.com/BananaDest/Chirpy/internal/database"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type apiConfig struct {
	fileServerHits  atomic.Int32
	databaseQueries *database.Queries
	platform        string
}
type User struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email     string    `json:"email"`
}

func main() {
	err := godotenv.Load()
	if err != nil {
		fmt.Println(err)
	}
	dbURL := os.Getenv("DB_URL")
	platform := os.Getenv("PLATFORM")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		fmt.Println(err)
	}
	dbQueries := database.New(db)
	cfg := apiConfig{
		fileServerHits:  atomic.Int32{},
		databaseQueries: dbQueries,
		platform:        platform,
	}
	serveMux := http.NewServeMux()
	server := &http.Server{Addr: "localhost:8080", Handler: serveMux}
	fileHandler := http.FileServer(http.Dir("."))
	strippedFileHandler := http.StripPrefix("/app", fileHandler)
	serveMux.Handle("/app/", cfg.middlewareMetricsInc(strippedFileHandler))
	serveMux.HandleFunc("GET /api/healthz", func(res http.ResponseWriter, req *http.Request) {
		res.Header().Add("Content-Type", "text/plain; charset=utf-8")
		res.WriteHeader(200)
		_, err := res.Write([]byte("OK"))
		if err != nil {
			fmt.Println("error")
		}
	})
	serveMux.HandleFunc("GET /admin/metrics", cfg.metricsHandler)
	serveMux.HandleFunc("POST /admin/reset", cfg.resetHandler)
	serveMux.HandleFunc("POST /api/validate_chirp", cfg.validateChirpHandler)
	serveMux.HandleFunc("POST /api/users", cfg.createUser)
	err = server.ListenAndServe()
	if err != nil {
		fmt.Println(err)
	}
}

func (cfg *apiConfig) createUser(res http.ResponseWriter, req *http.Request) {
	type parameters struct {
		Email string `json:"email"`
	}
	type error struct {
		Error string `json:"error"`
	}

	decoder := json.NewDecoder(req.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		log.Printf("Error decoding parameters: %s", err)
		res.WriteHeader(500)
		res.Header().Set("Content-Type", "application/json")
		errorBody := error{
			Error: fmt.Sprintf("%v", err),
		}
		errorData, _ := json.Marshal(errorBody)
		res.Write(errorData)
		return
	}
	user, err := cfg.databaseQueries.CreateUser(req.Context(), params.Email)
	if err != nil {
		log.Printf("Error creating User: %s", err)
		res.WriteHeader(500)
		res.Header().Set("Content-Type", "application/json")
		errorBody := error{
			Error: fmt.Sprintf("%v", err),
		}
		errorData, _ := json.Marshal(errorBody)
		res.Write(errorData)
		return

	}
	userParsed := User{
		ID:        user.ID,
		Email:     user.Email,
		UpdatedAt: user.UpdatedAt,
		CreatedAt: user.CreatedAt,
	}
	dat, err := json.Marshal(userParsed)
	if err != nil {
		log.Printf("Error marshalling JSON: %s", err)
		res.WriteHeader(500)
		res.Header().Set("Content-Type", "application/json")
		errorBody := error{
			Error: fmt.Sprintf("%v", err),
		}
		errorData, _ := json.Marshal(errorBody)
		res.Write(errorData)
		return
	}
	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(201)
	res.Write(dat)
}

func (cfg *apiConfig) metricsHandler(res http.ResponseWriter, req *http.Request) {
	res.Header().Add("Content-Type", "text/html")
	res.WriteHeader(200)

	response := fmt.Sprintf(`<html>
  <body>
    <h1>Welcome, Chirpy Admin</h1>
    <p>Chirpy has been visited %d times!</p>
  </body>
</html>`, cfg.fileServerHits.Load())
	_, err := res.Write([]byte(response))
	if err != nil {
		fmt.Println("error")
	}
}

func (cfg *apiConfig) resetHandler(res http.ResponseWriter, req *http.Request) {
	if cfg.platform != "dev" {
		res.WriteHeader(500)
		return
	}
	cfg.fileServerHits.Swap(0)
	err := cfg.databaseQueries.DeleteUsers(req.Context())
	if err != nil {

		log.Printf("Error truncating users: %s", err)
		res.WriteHeader(500)
		return
	}
	res.Header().Add("Content-Type", "text/plain; charset=utf-8")
	res.WriteHeader(200)
	response := fmt.Sprintf("Hits: %v", cfg.fileServerHits.Load())
	_, err = res.Write([]byte(response))
	if err != nil {
		fmt.Println("error")
	}
}

func (cfg *apiConfig) validateChirpHandler(res http.ResponseWriter, req *http.Request) {
	type parameters struct {
		Body string `json:"body"`
	}
	type error struct {
		Error string `json:"error"`
	}

	decoder := json.NewDecoder(req.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		// an error will be thrown if the JSON is invalid or has the wrong types
		// any missing fields will simply have their values in the struct set to their zero value
		log.Printf("Error decoding parameters: %s", err)
		res.WriteHeader(500)
		res.Header().Set("Content-Type", "application/json")
		errorBody := error{
			Error: fmt.Sprintf("%v", err),
		}
		errorData, _ := json.Marshal(errorBody)
		res.Write(errorData)
		return
	}
	if len(params.Body) > 140 {
		res.WriteHeader(400)
		res.Header().Set("Content-Type", "application/json")
		errorBody := error{
			Error: "body is greater than 140",
		}
		errorData, _ := json.Marshal(errorBody)
		res.Write(errorData)
		return

	}

	type returnVals struct {
		// the key will be the name of struct field unless you give it an explicit JSON tag
		CleanedBody string `json:"cleaned_body"`
	}
	respBody := returnVals{
		CleanedBody: CleanString(params.Body),
	}
	dat, err := json.Marshal(respBody)
	if err != nil {
		log.Printf("Error marshalling JSON: %s", err)
		res.WriteHeader(500)
		res.Header().Set("Content-Type", "application/json")
		errorBody := error{
			Error: fmt.Sprintf("%v", err),
		}
		errorData, _ := json.Marshal(errorBody)
		res.Write(errorData)
		return
	}
	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(200)
	res.Write(dat)
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileServerHits.Add(1)
		next.ServeHTTP(w, r)
	})
}
