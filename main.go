package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync/atomic"
)

type apiConfig struct {
	fileServerHits atomic.Int32
}

func main() {
	cfg := apiConfig{
		fileServerHits: atomic.Int32{},
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
	err := server.ListenAndServe()
	if err != nil {
		fmt.Println(err)
	}
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
	res.Header().Add("Content-Type", "text/plain; charset=utf-8")
	res.WriteHeader(200)
	cfg.fileServerHits.Swap(0)
	response := fmt.Sprintf("Hits: %v", cfg.fileServerHits.Load())
	_, err := res.Write([]byte(response))
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
