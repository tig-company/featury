package audit

import (
	"context"
	"crypto/rand"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/tig-company/featury/internal/models"
)

type metadataExtractor struct {
	requestIDGenerator func() string
}

// NewMetadataExtractor creates a new metadata extractor
func NewMetadataExtractor() MetadataExtractor {
	return &metadataExtractor{
		requestIDGenerator: generateRequestID,
	}
}

// ExtractMetadata extracts audit metadata from the context
func (m *metadataExtractor) ExtractMetadata(ctx context.Context) models.JSONB {
	metadata := make(models.JSONB)

	// Add timestamp
	metadata["timestamp"] = time.Now().Format(time.RFC3339)

	// Add request ID (generate if not present)
	if requestID := m.extractRequestID(ctx); requestID != "" {
		metadata["request_id"] = requestID
	} else {
		metadata["request_id"] = m.requestIDGenerator()
	}

	// Extract request information
	if requestInfo := m.ExtractRequestInfo(ctx); len(requestInfo) > 0 {
		for key, value := range requestInfo {
			metadata[key] = value
		}
	}

	return metadata
}

// ExtractUserInfo extracts user information from context
func (m *metadataExtractor) ExtractUserInfo(ctx context.Context) (userID uuid.UUID, exists bool) {
	// Try to extract user ID from context
	if uid, ok := ctx.Value("user_id").(uuid.UUID); ok {
		return uid, true
	}

	// Try string format
	if uidStr, ok := ctx.Value("user_id").(string); ok {
		if uid, err := uuid.Parse(uidStr); err == nil {
			return uid, true
		}
	}

	return uuid.Nil, false
}

// ExtractRequestInfo extracts request information from context
func (m *metadataExtractor) ExtractRequestInfo(ctx context.Context) models.JSONB {
	info := make(models.JSONB)

	// Extract client IP
	if clientIP, ok := ctx.Value("client_ip").(string); ok {
		info["client_ip"] = clientIP
	}

	// Extract user agent
	if userAgent, ok := ctx.Value("user_agent").(string); ok {
		info["user_agent"] = userAgent
	}

	// Extract request method
	if method, ok := ctx.Value("request_method").(string); ok {
		info["request_method"] = method
	}

	// Extract request path
	if path, ok := ctx.Value("request_path").(string); ok {
		info["request_path"] = path
	}

	// Extract request headers (selective)
	if headers, ok := ctx.Value("request_headers").(map[string]string); ok {
		safeHeaders := make(map[string]string)
		// Only include safe headers, exclude sensitive ones
		safeHeaderKeys := []string{"accept", "accept-language", "content-type"}
		for _, key := range safeHeaderKeys {
			if value, exists := headers[key]; exists {
				safeHeaders[key] = value
			}
		}
		if len(safeHeaders) > 0 {
			info["headers"] = safeHeaders
		}
	}

	// Extract API key info (without the actual key)
	if apiKeyID, ok := ctx.Value("api_key_id").(uuid.UUID); ok {
		info["api_key_id"] = apiKeyID.String()
	}

	return info
}

// extractRequestID extracts request ID from context
func (m *metadataExtractor) extractRequestID(ctx context.Context) string {
	if requestID, ok := ctx.Value("request_id").(string); ok {
		return requestID
	}

	// Try alternative context keys
	if xRequestID, ok := ctx.Value("x-request-id").(string); ok {
		return xRequestID
	}

	return ""
}

// generateRequestID generates a new request ID
func generateRequestID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to timestamp-based ID
		return fmt.Sprintf("req_%d", time.Now().UnixNano())
	}
	return fmt.Sprintf("req_%x", bytes)
}

// ContextKey represents keys used in context
type ContextKey string

const (
	// Context keys for audit metadata
	ContextKeyUserID        ContextKey = "user_id"
	ContextKeyClientIP      ContextKey = "client_ip"
	ContextKeyUserAgent     ContextKey = "user_agent"
	ContextKeyRequestMethod ContextKey = "request_method"
	ContextKeyRequestPath   ContextKey = "request_path"
	ContextKeyRequestID     ContextKey = "request_id"
	ContextKeyAPIKeyID      ContextKey = "api_key_id"
	ContextKeyRequestHeaders ContextKey = "request_headers"
)

// WithUserID adds user ID to context
func WithUserID(ctx context.Context, userID uuid.UUID) context.Context {
	return context.WithValue(ctx, ContextKeyUserID, userID)
}

// WithClientIP adds client IP to context
func WithClientIP(ctx context.Context, clientIP string) context.Context {
	return context.WithValue(ctx, ContextKeyClientIP, clientIP)
}

// WithUserAgent adds user agent to context
func WithUserAgent(ctx context.Context, userAgent string) context.Context {
	return context.WithValue(ctx, ContextKeyUserAgent, userAgent)
}

// WithRequestMethod adds request method to context
func WithRequestMethod(ctx context.Context, method string) context.Context {
	return context.WithValue(ctx, ContextKeyRequestMethod, method)
}

// WithRequestPath adds request path to context
func WithRequestPath(ctx context.Context, path string) context.Context {
	return context.WithValue(ctx, ContextKeyRequestPath, path)
}

// WithRequestID adds request ID to context
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, ContextKeyRequestID, requestID)
}

// WithAPIKeyID adds API key ID to context
func WithAPIKeyID(ctx context.Context, apiKeyID uuid.UUID) context.Context {
	return context.WithValue(ctx, ContextKeyAPIKeyID, apiKeyID)
}