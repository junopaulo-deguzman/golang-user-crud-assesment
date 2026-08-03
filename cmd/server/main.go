package main

import (
	"context"
	"database/sql"
	"golang-user-crud-assesment/internal/handler"
	"golang-user-crud-assesment/internal/middleware"
	"golang-user-crud-assesment/internal/repository"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	username := os.Getenv("BASIC_AUTH_USERNAME")
	password := os.Getenv("BASIC_AUTH_PASSWORD")

	dsn := os.Getenv("MYSQL_DSN")

	if username == "" || password == "" {
		log.Fatal("BASIC_AUTH_USERNAME and BASIC_AUTH_PASSWORD environment variables must be set")
	}

	if dsn == "" {
		log.Fatal("MYSQL_DSN environment variable must be set")
	}

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("failed to connect to the database: %v", err)
	}
	defer db.Close()

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	ctx, cancel := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)

	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("failed to ping the database: %v", err)
	}

	log.Println("Successfully connected to the database")

	mux := http.NewServeMux()
	h := handler.New(repository.NewUserRepository(db))

	mux.Handle(
		"POST /users",
		middleware.BasicAuth(username, password, http.HandlerFunc(h.CreateUser)),
	)
	mux.Handle(
		"GET /users/{id}",
		middleware.BasicAuth(username, password, http.HandlerFunc(h.GetUser)),
	)
	mux.Handle(
		"PUT /users/{id}",
		middleware.BasicAuth(username, password, http.HandlerFunc(h.UpdateUser)),
	)

	mux.Handle(
		"PATCH /users/{id}",
		middleware.BasicAuth(username, password, http.HandlerFunc(h.PatchUser)),
	)

	mux.Handle(
		"DELETE /users/{id}",
		middleware.BasicAuth(username, password, http.HandlerFunc(h.DeleteUser)),
	)

	log.Printf("Server has started on :8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}
