package main

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/tig-company/featury/internal/api"
	"github.com/tig-company/featury/internal/config"
)

func main() {
	cfg := config.Load()

	r := gin.Default()
	
	api.SetupRoutes(r)

	log.Printf("Starting featury API server on port %s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, r); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}