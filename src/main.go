package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/insighta-labs/src/config"
	"github.com/insighta-labs/src/handlers"
	"github.com/insighta-labs/src/services"
	"github.com/insighta-labs/src/store"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	ctx := context.Background()
	db, err := store.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database connection error: %v", err)
	}
	defer db.Close()

	profileService := services.NewProfileService(db)
	profileHandler := handlers.NewProfileHandler(profileService)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/profiles", profileHandler.Create)
	mux.HandleFunc("GET /api/profiles/{id}", profileHandler.GetByID)
	mux.HandleFunc("GET /api/profiles", profileHandler.List)
	mux.HandleFunc("DELETE /api/profiles/{id}", profileHandler.Delete)

	srv := &http.Server{
		Addr:        ":" + cfg.Port,
		Handler:     handlers.CORS(mux),
		ReadTimeout: 5 * time.Second,
		IdleTimeout: 20 * time.Second,
	}

	log.Printf("Server is running on :%s", cfg.Port)
	log.Fatal(srv.ListenAndServe())
}