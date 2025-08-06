package main

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/tig-company/featury/internal/api"
	"github.com/tig-company/featury/internal/config"
	"github.com/tig-company/featury/internal/database"
	"github.com/tig-company/featury/internal/repository"
	"github.com/tig-company/featury/internal/service"
)

func main() {
	cfg := config.Load()

	// Initialize database connection
	db, err := database.ConnectWithURL(cfg.DatabaseURL)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	// Run database migrations
	if err := database.Migrate(cfg.DatabaseURL); err != nil {
		log.Fatal("Failed to run database migrations:", err)
	}

	// Initialize repositories
	repo := repository.NewRepository(db)

	// Initialize services
	authService := service.NewAuthService(
		repo.APIKeys(),
		repo.Users(),
		repo.AuditLogs(),
	)

	// Set Gin mode based on environment
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	
	// Setup routes with authentication service
	api.SetupRoutes(r, authService)

	log.Printf("Starting featury API server on port %s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, r); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}