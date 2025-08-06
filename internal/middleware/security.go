package middleware

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// SecurityConfig contains configuration for security middleware
type SecurityConfig struct {
	// CORS settings
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	ExposeHeaders    []string
	AllowCredentials bool
	MaxAge           time.Duration

	// Security headers
	ContentTypeNoSniff    bool
	FrameOptions          string // DENY, SAMEORIGIN, or ALLOW-FROM uri
	XSSProtection         string // "1; mode=block" or "0"
	ContentSecurityPolicy string
	ReferrerPolicy        string
	
	// Custom headers
	CustomHeaders map[string]string
	
	// Request ID settings
	RequestIDHeader string
	GenerateRequestID bool
}

// DefaultSecurityConfig returns default security configuration
func DefaultSecurityConfig() *SecurityConfig {
	return &SecurityConfig{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"},
		AllowedHeaders: []string{
			"Origin",
			"Content-Length",
			"Content-Type",
			"Authorization",
			"Accept",
			"X-Requested-With",
			"X-Request-ID",
		},
		ExposeHeaders: []string{
			"X-Request-ID",
			"X-RateLimit-Limited",
			"Retry-After",
		},
		AllowCredentials:      false,
		MaxAge:               12 * time.Hour,
		ContentTypeNoSniff:    true,
		FrameOptions:          "DENY",
		XSSProtection:         "1; mode=block",
		ContentSecurityPolicy: "default-src 'self'",
		ReferrerPolicy:        "strict-origin-when-cross-origin",
		RequestIDHeader:       "X-Request-ID",
		GenerateRequestID:     true,
		CustomHeaders: map[string]string{
			"X-Content-Type-Options": "nosniff",
			"X-Download-Options":     "noopen",
			"X-Permitted-Cross-Domain-Policies": "none",
		},
	}
}

// ProductionSecurityConfig returns security configuration optimized for production
func ProductionSecurityConfig(allowedOrigins []string) *SecurityConfig {
	config := DefaultSecurityConfig()
	
	// Restrict origins in production
	if len(allowedOrigins) > 0 {
		config.AllowedOrigins = allowedOrigins
	}
	
	// Stricter security headers for production
	config.ContentSecurityPolicy = "default-src 'self'; script-src 'self'; object-src 'none'; base-uri 'self';"
	config.CustomHeaders["Strict-Transport-Security"] = "max-age=31536000; includeSubDomains"
	
	return config
}

// SecurityMiddleware manages security headers and CORS
type SecurityMiddleware struct {
	config *SecurityConfig
}

// NewSecurityMiddleware creates a new security middleware
func NewSecurityMiddleware(config *SecurityConfig) *SecurityMiddleware {
	if config == nil {
		config = DefaultSecurityConfig()
	}
	return &SecurityMiddleware{config: config}
}

// CORS returns middleware that handles Cross-Origin Resource Sharing
func (sm *SecurityMiddleware) CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		
		// Check if origin is allowed
		if sm.isOriginAllowed(origin) {
			c.Header("Access-Control-Allow-Origin", origin)
		} else if len(sm.config.AllowedOrigins) == 1 && sm.config.AllowedOrigins[0] == "*" {
			c.Header("Access-Control-Allow-Origin", "*")
		}

		// Set other CORS headers
		if len(sm.config.AllowedMethods) > 0 {
			c.Header("Access-Control-Allow-Methods", strings.Join(sm.config.AllowedMethods, ", "))
		}
		
		if len(sm.config.AllowedHeaders) > 0 {
			c.Header("Access-Control-Allow-Headers", strings.Join(sm.config.AllowedHeaders, ", "))
		}
		
		if len(sm.config.ExposeHeaders) > 0 {
			c.Header("Access-Control-Expose-Headers", strings.Join(sm.config.ExposeHeaders, ", "))
		}
		
		if sm.config.AllowCredentials {
			c.Header("Access-Control-Allow-Credentials", "true")
		}
		
		if sm.config.MaxAge > 0 {
			c.Header("Access-Control-Max-Age", string(rune(int(sm.config.MaxAge.Seconds()))))
		}

		// Handle preflight OPTIONS request
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// SecurityHeaders returns middleware that adds security headers
func (sm *SecurityMiddleware) SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Content-Type sniffing protection
		if sm.config.ContentTypeNoSniff {
			c.Header("X-Content-Type-Options", "nosniff")
		}

		// Frame options (clickjacking protection)
		if sm.config.FrameOptions != "" {
			c.Header("X-Frame-Options", sm.config.FrameOptions)
		}

		// XSS protection
		if sm.config.XSSProtection != "" {
			c.Header("X-XSS-Protection", sm.config.XSSProtection)
		}

		// Content Security Policy
		if sm.config.ContentSecurityPolicy != "" {
			c.Header("Content-Security-Policy", sm.config.ContentSecurityPolicy)
		}

		// Referrer Policy
		if sm.config.ReferrerPolicy != "" {
			c.Header("Referrer-Policy", sm.config.ReferrerPolicy)
		}

		// Custom headers
		for key, value := range sm.config.CustomHeaders {
			c.Header(key, value)
		}

		c.Next()
	}
}

// RequestID returns middleware that generates and sets request IDs
func (sm *SecurityMiddleware) RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		var requestID string

		// Check if request ID is already provided
		if sm.config.RequestIDHeader != "" {
			requestID = c.GetHeader(sm.config.RequestIDHeader)
		}

		// Generate new request ID if needed
		if requestID == "" && sm.config.GenerateRequestID {
			requestID = generateRequestID()
		}

		// Set request ID in context and response header
		if requestID != "" {
			c.Set("RequestID", requestID)
			if sm.config.RequestIDHeader != "" {
				c.Header(sm.config.RequestIDHeader, requestID)
			}
		}

		c.Next()
	}
}

// Logger returns middleware for request logging with security considerations
func (sm *SecurityMiddleware) Logger() gin.HandlerFunc {
	return gin.LoggerWithConfig(gin.LoggerConfig{
		Formatter: func(param gin.LogFormatterParams) string {
			// Custom log format that includes request ID and sanitizes sensitive data
			requestID := ""
			if param.Keys != nil {
				if id, exists := param.Keys["RequestID"]; exists {
					if idStr, ok := id.(string); ok {
						requestID = idStr
					}
				}
			}

			// Sanitize the path to prevent log injection
			path := sanitizeLogPath(param.Path)
			
			return fmt.Sprintf("[%s] %s %3d %13v %15s %s %-7s %s %s\n",
				param.TimeStamp.Format("2006/01/02 - 15:04:05"),
				requestID,
				param.StatusCode,
				param.Latency,
				param.ClientIP,
				param.Method,
				path,
				param.ErrorMessage,
			)
		},
		SkipPaths: []string{"/health", "/ping"}, // Skip logging for health check endpoints
	})
}

// CombinedSecurityMiddleware returns a combined middleware that applies all security measures
func (sm *SecurityMiddleware) CombinedSecurityMiddleware() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		// Apply request ID first
		if sm.config.GenerateRequestID {
			sm.RequestID()(c)
		}

		// Apply CORS
		sm.CORS()(c)
		
		// Apply security headers
		sm.SecurityHeaders()(c)

		c.Next()
	})
}

// Helper methods

// isOriginAllowed checks if an origin is in the allowed origins list
func (sm *SecurityMiddleware) isOriginAllowed(origin string) bool {
	if origin == "" {
		return false
	}

	for _, allowedOrigin := range sm.config.AllowedOrigins {
		if allowedOrigin == "*" {
			return true
		}
		if allowedOrigin == origin {
			return true
		}
		// Support wildcard subdomains (e.g., "*.example.com")
		if strings.HasPrefix(allowedOrigin, "*.") {
			domain := strings.TrimPrefix(allowedOrigin, "*")
			if strings.HasSuffix(origin, domain) {
				return true
			}
		}
	}

	return false
}

// generateRequestID generates a new UUID-based request ID
func generateRequestID() string {
	return uuid.New().String()
}

// sanitizeLogPath sanitizes a path for safe logging
func sanitizeLogPath(path string) string {
	// Remove newlines and other control characters that could be used for log injection
	path = strings.ReplaceAll(path, "\n", "")
	path = strings.ReplaceAll(path, "\r", "")
	path = strings.ReplaceAll(path, "\t", "")
	
	// Limit path length to prevent log flooding
	if len(path) > 200 {
		path = path[:200] + "..."
	}
	
	return path
}

// ValidateSecurityConfig validates the security configuration
func ValidateSecurityConfig(config *SecurityConfig) error {
	if config == nil {
		return fmt.Errorf("security config cannot be nil")
	}

	// Validate frame options
	validFrameOptions := []string{"DENY", "SAMEORIGIN"}
	if config.FrameOptions != "" {
		valid := false
		for _, option := range validFrameOptions {
			if config.FrameOptions == option {
				valid = true
				break
			}
		}
		// Also allow ALLOW-FROM format
		if !valid && strings.HasPrefix(config.FrameOptions, "ALLOW-FROM ") {
			valid = true
		}
		if !valid {
			return fmt.Errorf("invalid frame options: %s", config.FrameOptions)
		}
	}

	// Validate XSS protection
	if config.XSSProtection != "" {
		validXSS := []string{"0", "1", "1; mode=block"}
		valid := false
		for _, option := range validXSS {
			if config.XSSProtection == option {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("invalid XSS protection: %s", config.XSSProtection)
		}
	}

	return nil
}