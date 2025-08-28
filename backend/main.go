package main

import (
	"encoding/json"
	"log"
	"net/http"
)

type healthResponse struct {
	Status  string `json:"status"`
	Service string `json:"service"`
	Version string `json:"version"`
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(healthResponse{
		Status:  "ok",
		Service: "cratedrop-backend",
		Version: "v0",
	})
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/healthz", healthHandler)
	mux.HandleFunc("/healthz", healthHandler)

	addr := ":8080"
	log.Printf("Starting music server on %s...", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
