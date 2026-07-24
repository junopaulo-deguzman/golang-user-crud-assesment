package main

import (
	"encoding/json"
	"golang-user-crud-assesment/internal/handler"
	"golang-user-crud-assesment/internal/middleware"
	"log"
	"net/http"
	"os"
)

func main() {
	username := os.Getenv("BASIC_AUTH_USERNAME")
	password := os.Getenv("BASIC_AUTH_PASSWORD")

	if username == "" || password == "" {
		log.Fatal("BASIC_AUTH_USERNAME and BASIC_AUTH_PASSWORD environment variables must be set")
	}

	mux := http.NewServeMux()
	h := &handler.Handler{}

	mux.HandleFunc("GET /health", health)
	mux.Handle(
		"POST /users",
		middleware.BasicAuth(username, password, http.HandlerFunc(h.CreateUser)),
	)
	mux.Handle(
		"GET /users",
		middleware.BasicAuth(username, password, http.HandlerFunc(h.GetUser)),
	)

	log.Printf("Server has started on :8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}

}

func health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("failed to write json response: %v", err)
	}
}
