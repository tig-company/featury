package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// ResponseMiddleware provides consistent response formatting and headers
func ResponseMiddleware() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		// Set common response headers
		c.Header("Content-Type", "application/json")
		c.Header("X-Service", "featury")
		c.Header("X-Version", "1.0.0")
		c.Header("X-Request-ID", getRequestID(c))
		c.Header("X-Response-Time", time.Now().Format(time.RFC3339))
		
		// CORS headers (if not handled elsewhere)
		if origin := c.GetHeader("Origin"); origin != "" {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization, X-API-Key")
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Access-Control-Max-Age", "86400")
		}
		
		// Handle preflight requests
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		
		// Create a custom response writer to capture response details
		writer := &responseWriter{
			ResponseWriter: c.Writer,
			startTime:      time.Now(),
		}
		c.Writer = writer
		
		// Process request
		c.Next()
		
		// Add response time header after processing
		duration := time.Since(writer.startTime)
		c.Header("X-Response-Time-Ms", strconv.FormatInt(duration.Milliseconds(), 10))
		
		// Log response details (could be enhanced with structured logging)
		logResponseDetails(c, writer, duration)
	})
}

// responseWriter wraps gin.ResponseWriter to capture response details
type responseWriter struct {
	gin.ResponseWriter
	startTime time.Time
	size      int
}

// Write captures the response size
func (w *responseWriter) Write(data []byte) (int, error) {
	size, err := w.ResponseWriter.Write(data)
	w.size += size
	return size, err
}

// WriteString captures the response size for string writes
func (w *responseWriter) WriteString(s string) (int, error) {
	size, err := w.ResponseWriter.WriteString(s)
	w.size += size
	return size, err
}

// JSONResponseMiddleware ensures consistent JSON response structure
func JSONResponseMiddleware() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		c.Next()
		
		// Only process if no response has been written and there are no errors
		if c.Writer.Written() || len(c.Errors) > 0 {
			return
		}
		
		// This middleware doesn't modify existing responses
		// It just ensures JSON content type is set
		if c.Writer.Header().Get("Content-Type") == "" {
			c.Header("Content-Type", "application/json")
		}
	})
}

// PaginationResponseMiddleware adds pagination metadata to responses
func PaginationResponseMiddleware() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		c.Next()
		
		// Add pagination-related headers if pagination parameters were provided
		if page := c.Query("page"); page != "" {
			c.Header("X-Page", page)
		}
		
		if limit := c.Query("limit"); limit != "" {
			c.Header("X-Limit", limit)
		}
		
		// These headers would be set by the handlers that support pagination
		// We're just ensuring they're consistently formatted
		if totalCount := c.Writer.Header().Get("X-Total-Count"); totalCount != "" {
			// Total count header is already set by handler
		}
		
		if totalPages := c.Writer.Header().Get("X-Total-Pages"); totalPages != "" {
			// Total pages header is already set by handler
		}
	})
}

// CacheControlMiddleware adds appropriate cache control headers
func CacheControlMiddleware() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		// Set cache control based on request method and endpoint
		switch c.Request.Method {
		case "GET":
			// For read operations, allow short caching
			if isStaticEndpoint(c.FullPath()) {
				c.Header("Cache-Control", "public, max-age=300") // 5 minutes
			} else {
				c.Header("Cache-Control", "private, max-age=60") // 1 minute for dynamic content
			}
		case "POST", "PUT", "PATCH", "DELETE":
			// For write operations, prevent caching
			c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
			c.Header("Pragma", "no-cache")
			c.Header("Expires", "0")
		default:
			c.Header("Cache-Control", "no-cache")
		}
		
		c.Next()
	})
}

// SecurityHeadersMiddleware adds security-related headers
func SecurityHeadersMiddleware() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		// Security headers
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("Content-Security-Policy", "default-src 'self'")
		
		// Remove server information
		c.Header("Server", "")
		
		c.Next()
	})
}

// CompressionMiddleware would handle response compression
// Note: Gin has built-in gzip middleware, this is just a placeholder for custom compression logic
func CompressionMiddleware() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		// This would implement custom compression logic if needed
		// For now, we'll rely on Gin's built-in gzip middleware
		c.Next()
	})
}

// Helper functions

// getRequestID extracts or generates a request ID
func getRequestID(c *gin.Context) string {
	// Try to get request ID from context (set by earlier middleware)
	if requestID, exists := c.Get("RequestID"); exists {
		if id, ok := requestID.(string); ok {
			return id
		}
	}
	
	// Try to get from header
	if requestID := c.GetHeader("X-Request-ID"); requestID != "" {
		return requestID
	}
	
	// Generate a simple request ID (in production, you'd want a more sophisticated approach)
	return "req-" + strconv.FormatInt(time.Now().UnixNano(), 36)
}

// isStaticEndpoint determines if an endpoint serves relatively static content
func isStaticEndpoint(path string) bool {
	staticPaths := []string{
		"/health",
		"/ping",
		"/metrics",
		"/environments",
		"/services",
	}
	
	for _, staticPath := range staticPaths {
		if path == staticPath {
			return true
		}
	}
	
	return false
}

// logResponseDetails logs response information (placeholder for structured logging)
func logResponseDetails(c *gin.Context, writer *responseWriter, duration time.Duration) {
	// In a production environment, you'd want to use a structured logger
	// This is just a placeholder to show what information would be logged
	
	status := c.Writer.Status()
	method := c.Request.Method
	path := c.Request.URL.Path
	userAgent := c.Request.UserAgent()
	clientIP := c.ClientIP()
	
	// These would be logged to your structured logger
	_ = status
	_ = method
	_ = path
	_ = userAgent
	_ = clientIP
	_ = duration
	_ = writer.size
	
	// Example structured log entry (commented out as we don't have a logger configured here):
	// log.Info("HTTP Request",
	//     "method", method,
	//     "path", path,
	//     "status", status,
	//     "duration_ms", duration.Milliseconds(),
	//     "response_size", writer.size,
	//     "client_ip", clientIP,
	//     "user_agent", userAgent,
	//     "request_id", getRequestID(c),
	// )
}