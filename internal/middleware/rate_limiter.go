package middleware

import (
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/tig-company/featury/pkg/errors"
)

// TokenBucket implements a token bucket rate limiter
type TokenBucket struct {
	tokens    int
	capacity  int
	refillRate int        // tokens per second
	lastRefill time.Time
	mutex     sync.RWMutex
}

// NewTokenBucket creates a new token bucket
func NewTokenBucket(capacity, refillRate int) *TokenBucket {
	return &TokenBucket{
		tokens:     capacity,
		capacity:   capacity,
		refillRate: refillRate,
		lastRefill: time.Now(),
	}
}

// TryConsume attempts to consume a token from the bucket
func (tb *TokenBucket) TryConsume() bool {
	tb.mutex.Lock()
	defer tb.mutex.Unlock()

	// Refill tokens based on elapsed time
	now := time.Now()
	elapsed := now.Sub(tb.lastRefill)
	
	if elapsed > 0 {
		tokensToAdd := int(elapsed.Seconds()) * tb.refillRate
		tb.tokens = min(tb.capacity, tb.tokens+tokensToAdd)
		tb.lastRefill = now
	}

	// Try to consume a token
	if tb.tokens > 0 {
		tb.tokens--
		return true
	}

	return false
}

// GetTokens returns the current number of tokens
func (tb *TokenBucket) GetTokens() int {
	tb.mutex.RLock()
	defer tb.mutex.RUnlock()
	return tb.tokens
}

// TimeToRefill returns the time until the next token is added
func (tb *TokenBucket) TimeToRefill() time.Duration {
	tb.mutex.RLock()
	defer tb.mutex.RUnlock()
	
	if tb.tokens >= tb.capacity {
		return 0
	}
	
	return time.Second / time.Duration(tb.refillRate)
}

// RateLimiterConfig contains configuration for rate limiting
type RateLimiterConfig struct {
	// Global rate limits (applied to all requests)
	GlobalRequestsPerSecond int
	GlobalBurstSize        int
	
	// Per-API-key rate limits
	APIKeyRequestsPerSecond int
	APIKeyBurstSize        int
	
	// Per-IP rate limits (fallback for unauthenticated requests)
	IPRequestsPerSecond int
	IPBurstSize        int
	
	// Cleanup settings
	CleanupInterval    time.Duration
	BucketExpireAfter  time.Duration
	
	// Enable/disable different rate limiting strategies
	EnableGlobalRateLimit bool
	EnableAPIKeyRateLimit bool
	EnableIPRateLimit     bool
}

// DefaultRateLimiterConfig returns default configuration
func DefaultRateLimiterConfig() *RateLimiterConfig {
	return &RateLimiterConfig{
		GlobalRequestsPerSecond: 1000,
		GlobalBurstSize:        200,
		APIKeyRequestsPerSecond: 100,
		APIKeyBurstSize:        20,
		IPRequestsPerSecond:    10,
		IPBurstSize:           5,
		CleanupInterval:       5 * time.Minute,
		BucketExpireAfter:     10 * time.Minute,
		EnableGlobalRateLimit: true,
		EnableAPIKeyRateLimit: true,
		EnableIPRateLimit:     true,
	}
}

// RateLimiter manages rate limiting using token buckets
type RateLimiter struct {
	config *RateLimiterConfig
	
	// Different bucket stores for different rate limiting strategies
	globalBucket    *TokenBucket
	apiKeyBuckets   map[uuid.UUID]*TokenBucket
	ipBuckets       map[string]*TokenBucket
	
	// Tracking last access time for cleanup
	apiKeyLastAccess map[uuid.UUID]time.Time
	ipLastAccess     map[string]time.Time
	
	mutex sync.RWMutex
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(config *RateLimiterConfig) *RateLimiter {
	if config == nil {
		config = DefaultRateLimiterConfig()
	}

	rl := &RateLimiter{
		config:           config,
		apiKeyBuckets:    make(map[uuid.UUID]*TokenBucket),
		ipBuckets:        make(map[string]*TokenBucket),
		apiKeyLastAccess: make(map[uuid.UUID]time.Time),
		ipLastAccess:     make(map[string]time.Time),
	}

	// Create global bucket if enabled
	if config.EnableGlobalRateLimit {
		rl.globalBucket = NewTokenBucket(config.GlobalBurstSize, config.GlobalRequestsPerSecond)
	}

	// Start cleanup goroutine
	go rl.cleanupExpiredBuckets()

	return rl
}

// getOrCreateAPIKeyBucket gets or creates a token bucket for an API key
func (rl *RateLimiter) getOrCreateAPIKeyBucket(apiKeyID uuid.UUID) *TokenBucket {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	bucket, exists := rl.apiKeyBuckets[apiKeyID]
	if !exists {
		bucket = NewTokenBucket(rl.config.APIKeyBurstSize, rl.config.APIKeyRequestsPerSecond)
		rl.apiKeyBuckets[apiKeyID] = bucket
	}
	
	rl.apiKeyLastAccess[apiKeyID] = time.Now()
	return bucket
}

// getOrCreateIPBucket gets or creates a token bucket for an IP address
func (rl *RateLimiter) getOrCreateIPBucket(ip string) *TokenBucket {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	bucket, exists := rl.ipBuckets[ip]
	if !exists {
		bucket = NewTokenBucket(rl.config.IPBurstSize, rl.config.IPRequestsPerSecond)
		rl.ipBuckets[ip] = bucket
	}
	
	rl.ipLastAccess[ip] = time.Now()
	return bucket
}

// checkRateLimit checks if a request should be rate limited
func (rl *RateLimiter) checkRateLimit(c *gin.Context) (bool, time.Duration) {
	// Check global rate limit first
	if rl.config.EnableGlobalRateLimit && rl.globalBucket != nil {
		if !rl.globalBucket.TryConsume() {
			return true, rl.globalBucket.TimeToRefill()
		}
	}

	// Try to get API key from context
	if apiKey, exists := GetAPIKeyFromContext(c); exists && rl.config.EnableAPIKeyRateLimit {
		bucket := rl.getOrCreateAPIKeyBucket(apiKey.ID)
		if !bucket.TryConsume() {
			return true, bucket.TimeToRefill()
		}
	} else if rl.config.EnableIPRateLimit {
		// Fall back to IP-based rate limiting for unauthenticated requests
		clientIP := getClientIP(c)
		if clientIP != "" {
			bucket := rl.getOrCreateIPBucket(clientIP)
			if !bucket.TryConsume() {
				return true, bucket.TimeToRefill()
			}
		}
	}

	return false, 0
}

// cleanupExpiredBuckets removes unused token buckets to prevent memory leaks
func (rl *RateLimiter) cleanupExpiredBuckets() {
	ticker := time.NewTicker(rl.config.CleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()
		
		rl.mutex.Lock()
		
		// Clean up expired API key buckets
		for apiKeyID, lastAccess := range rl.apiKeyLastAccess {
			if now.Sub(lastAccess) > rl.config.BucketExpireAfter {
				delete(rl.apiKeyBuckets, apiKeyID)
				delete(rl.apiKeyLastAccess, apiKeyID)
			}
		}
		
		// Clean up expired IP buckets
		for ip, lastAccess := range rl.ipLastAccess {
			if now.Sub(lastAccess) > rl.config.BucketExpireAfter {
				delete(rl.ipBuckets, ip)
				delete(rl.ipLastAccess, ip)
			}
		}
		
		rl.mutex.Unlock()
	}
}

// RateLimit returns a Gin middleware that enforces rate limits
func (rl *RateLimiter) RateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		limited, retryAfter := rl.checkRateLimit(c)
		
		if limited {
			// Add rate limit headers
			c.Header("X-RateLimit-Limited", "true")
			if retryAfter > 0 {
				c.Header("Retry-After", fmt.Sprintf("%d", int(retryAfter.Seconds())))
			}
			
			errors.AbortWithRateLimit(c, retryAfter)
			return
		}

		// Add rate limit headers for successful requests
		c.Header("X-RateLimit-Limited", "false")
		
		c.Next()
	}
}

// GetStats returns current rate limiting statistics
func (rl *RateLimiter) GetStats() map[string]interface{} {
	rl.mutex.RLock()
	defer rl.mutex.RUnlock()

	stats := map[string]interface{}{
		"total_api_key_buckets": len(rl.apiKeyBuckets),
		"total_ip_buckets":      len(rl.ipBuckets),
		"global_tokens":         0,
	}

	if rl.globalBucket != nil {
		stats["global_tokens"] = rl.globalBucket.GetTokens()
	}

	return stats
}

// getClientIP extracts the real client IP address from the request
func getClientIP(c *gin.Context) string {
	// Check X-Forwarded-For header first
	xForwardedFor := c.GetHeader("X-Forwarded-For")
	if xForwardedFor != "" {
		// X-Forwarded-For can contain multiple IPs, take the first one
		ips := parseForwardedFor(xForwardedFor)
		if len(ips) > 0 {
			return ips[0]
		}
	}

	// Check X-Real-IP header
	xRealIP := c.GetHeader("X-Real-IP")
	if xRealIP != "" {
		return xRealIP
	}

	// Fall back to RemoteAddr
	ip, _, err := net.SplitHostPort(c.Request.RemoteAddr)
	if err != nil {
		return c.Request.RemoteAddr
	}
	return ip
}

// parseForwardedFor parses the X-Forwarded-For header
func parseForwardedFor(header string) []string {
	var ips []string
	for _, ip := range strings.Split(header, ",") {
		ip = strings.TrimSpace(ip)
		if ip != "" {
			ips = append(ips, ip)
		}
	}
	return ips
}

// Helper function to get minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}