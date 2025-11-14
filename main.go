package main

import (
	"fmt"
	"net/http"
)

func main() {
	serveMux := http.NewServeMux()
	server := &http.Server{Addr: "localhost:8080", Handler: serveMux}
	fileHandler := http.FileServer(http.Dir("."))
	serveMux.Handle("/", fileHandler)
	err := server.ListenAndServe()
	if err != nil {
		fmt.Println(err)
	}
}
