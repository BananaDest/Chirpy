package main

import (
	"fmt"
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
	serveMux.HandleFunc("/healthz", func(res http.ResponseWriter, req *http.Request) {
		res.Header().Add("Content-Type", "text/plain; charset=utf-8")
		res.WriteHeader(200)
		_, err := res.Write([]byte("OK"))
		if err != nil {
			fmt.Println("error")
		}
	})
	serveMux.HandleFunc("/metrics", cfg.metricsHandler)
	serveMux.HandleFunc("/reset", cfg.resetHandler)
	err := server.ListenAndServe()
	if err != nil {
		fmt.Println(err)
	}
}

func (cfg *apiConfig) metricsHandler(res http.ResponseWriter, req *http.Request) {
	res.Header().Add("Content-Type", "text/plain; charset=utf-8")
	res.WriteHeader(200)
	response := fmt.Sprintf("Hits: %v", cfg.fileServerHits.Load())
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

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileServerHits.Add(1)
		next.ServeHTTP(w, r)
	})
}
