package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tig-company/featury/internal/api/routes"
	"github.com/tig-company/featury/internal/audit"
	"github.com/tig-company/featury/internal/cache"
	"github.com/tig-company/featury/internal/config"
	"github.com/tig-company/featury/internal/database"
	"github.com/tig-company/featury/internal/middleware"
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

	// Initialize cache service (using memory cache as fallback)
	cacheMetrics := cache.NewCacheMetrics()
	memCache := cache.NewMemoryCache(cacheMetrics)
	
	// Create adapter for service.CacheService interface
	cacheService := &cacheServiceAdapter{cache: memCache}

	// Initialize audit components
	differ := audit.NewObjectDiffer()
	metadataExtractor := audit.NewMetadataExtractor()
	auditTracker := audit.NewAuditTracker(repo, differ, metadataExtractor)

	// Initialize services
	authService := service.NewAuthService(
		repo.APIKeys(),
		repo.Users(),
		repo.AuditLogs(),
	)

	auditService := service.NewAuditService(repo, auditTracker, differ)
	diffService := service.NewDiffService(differ)
	validationService := service.NewValidationService(repo)
	
	featureFlagService := service.NewFeatureFlagService(
		repo,
		validationService,
		auditService,
		diffService,
	)

	// Create middleware stack
	middlewareStack := middleware.NewMiddlewareStack(nil, authService)

	// Set Gin mode based on environment
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	
	// Setup routes with new architecture
	routerConfig := &routes.RouterConfig{
		DB:                 db,
		MiddlewareStack:    middlewareStack,
		AuthService:        authService,
		FeatureFlagService: featureFlagService,
		CacheService:       cacheService,
		AuditService:       auditService,
	}
	
	routes.SetupRoutes(r, routerConfig)

	log.Printf("Starting featury API server on port %s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, r); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}

// cacheServiceAdapter adapts cache.Cache to service.CacheService interface
type cacheServiceAdapter struct {
	cache cache.Cache
}

func (a *cacheServiceAdapter) Get(ctx context.Context, key string) ([]byte, error) {
	return a.cache.Get(ctx, key)
}

func (a *cacheServiceAdapter) Set(ctx context.Context, key string, value []byte, ttl int) error {
	// Convert TTL from int (seconds) to time.Duration
	return a.cache.Set(ctx, key, value, time.Duration(ttl)*time.Second)
}

func (a *cacheServiceAdapter) Delete(ctx context.Context, key string) error {
	return a.cache.Delete(ctx, key)
}

func (a *cacheServiceAdapter) DeletePattern(ctx context.Context, pattern string) error {
	return a.cache.DeletePattern(ctx, pattern)
}

func (a *cacheServiceAdapter) Exists(ctx context.Context, key string) (bool, error) {
	return a.cache.Exists(ctx, key)
}

func (a *cacheServiceAdapter) Health(ctx context.Context) error {
	return a.cache.Health(ctx)
}