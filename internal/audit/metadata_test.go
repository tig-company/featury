package audit

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMetadataExtractor(t *testing.T) {
	extractor := NewMetadataExtractor()

	t.Run("Basic Metadata Extraction", func(t *testing.T) {
		ctx := context.Background()
		ctx = context.WithValue(ctx, ContextKeyClientIP, "192.168.1.1")
		ctx = context.WithValue(ctx, ContextKeyUserAgent, "Mozilla/5.0")
		ctx = context.WithValue(ctx, ContextKeyRequestMethod, "POST")
		ctx = context.WithValue(ctx, ContextKeyRequestPath, "/api/flags")

		metadata := extractor.ExtractMetadata(ctx)

		assert.Contains(t, metadata, "timestamp")
		assert.Contains(t, metadata, "request_id")
		assert.Equal(t, "192.168.1.1", metadata["client_ip"])
		assert.Equal(t, "Mozilla/5.0", metadata["user_agent"])
		assert.Equal(t, "POST", metadata["request_method"])
		assert.Equal(t, "/api/flags", metadata["request_path"])

		// Verify timestamp format
		timestampStr := metadata["timestamp"].(string)
		_, err := time.Parse(time.RFC3339, timestampStr)
		require.NoError(t, err)

		// Verify request ID format
		requestID := metadata["request_id"].(string)
		assert.NotEmpty(t, requestID)
	})

	t.Run("User Info Extraction", func(t *testing.T) {
		userID := uuid.New()
		ctx := context.Background()
		ctx = context.WithValue(ctx, ContextKeyUserID, userID)

		extractedUserID, exists := extractor.ExtractUserInfo(ctx)
		assert.True(t, exists)
		assert.Equal(t, userID, extractedUserID)
	})

	t.Run("User Info Extraction - String Format", func(t *testing.T) {
		userID := uuid.New()
		ctx := context.Background()
		ctx = context.WithValue(ctx, ContextKeyUserID, userID.String())

		extractedUserID, exists := extractor.ExtractUserInfo(ctx)
		assert.True(t, exists)
		assert.Equal(t, userID, extractedUserID)
	})

	t.Run("User Info Not Found", func(t *testing.T) {
		ctx := context.Background()

		_, exists := extractor.ExtractUserInfo(ctx)
		assert.False(t, exists)
	})

	t.Run("Request Info with Headers", func(t *testing.T) {
		headers := map[string]string{
			"accept":          "application/json",
			"accept-language": "en-US",
			"content-type":    "application/json",
			"authorization":   "Bearer token123", // Should be filtered out
		}

		ctx := context.Background()
		ctx = context.WithValue(ctx, ContextKeyRequestHeaders, headers)

		requestInfo := extractor.ExtractRequestInfo(ctx)

		safeHeaders := requestInfo["headers"].(map[string]string)
		assert.Equal(t, "application/json", safeHeaders["accept"])
		assert.Equal(t, "en-US", safeHeaders["accept-language"])
		assert.Equal(t, "application/json", safeHeaders["content-type"])
		
		// Authorization header should not be included
		assert.NotContains(t, safeHeaders, "authorization")
	})

	t.Run("API Key Info", func(t *testing.T) {
		apiKeyID := uuid.New()
		ctx := context.Background()
		ctx = context.WithValue(ctx, ContextKeyAPIKeyID, apiKeyID)

		requestInfo := extractor.ExtractRequestInfo(ctx)

		assert.Equal(t, apiKeyID.String(), requestInfo["api_key_id"])
	})

	t.Run("Request ID from Context", func(t *testing.T) {
		requestID := "req_12345"
		ctx := context.Background()
		ctx = context.WithValue(ctx, ContextKeyRequestID, requestID)

		metadata := extractor.ExtractMetadata(ctx)
		assert.Equal(t, requestID, metadata["request_id"])
	})

	t.Run("Request ID Generation", func(t *testing.T) {
		ctx := context.Background()

		metadata := extractor.ExtractMetadata(ctx)
		requestID := metadata["request_id"].(string)
		
		assert.NotEmpty(t, requestID)
		assert.Contains(t, requestID, "req_")
	})
}

func TestContextHelpers(t *testing.T) {
	t.Run("WithUserID", func(t *testing.T) {
		userID := uuid.New()
		ctx := WithUserID(context.Background(), userID)

		extractedUserID := ctx.Value(ContextKeyUserID).(uuid.UUID)
		assert.Equal(t, userID, extractedUserID)
	})

	t.Run("WithClientIP", func(t *testing.T) {
		clientIP := "10.0.0.1"
		ctx := WithClientIP(context.Background(), clientIP)

		extractedIP := ctx.Value(ContextKeyClientIP).(string)
		assert.Equal(t, clientIP, extractedIP)
	})

	t.Run("WithUserAgent", func(t *testing.T) {
		userAgent := "Test Agent/1.0"
		ctx := WithUserAgent(context.Background(), userAgent)

		extractedUA := ctx.Value(ContextKeyUserAgent).(string)
		assert.Equal(t, userAgent, extractedUA)
	})

	t.Run("WithRequestMethod", func(t *testing.T) {
		method := "PATCH"
		ctx := WithRequestMethod(context.Background(), method)

		extractedMethod := ctx.Value(ContextKeyRequestMethod).(string)
		assert.Equal(t, method, extractedMethod)
	})

	t.Run("WithRequestPath", func(t *testing.T) {
		path := "/api/v1/flags/123"
		ctx := WithRequestPath(context.Background(), path)

		extractedPath := ctx.Value(ContextKeyRequestPath).(string)
		assert.Equal(t, path, extractedPath)
	})

	t.Run("WithRequestID", func(t *testing.T) {
		requestID := "custom_req_id"
		ctx := WithRequestID(context.Background(), requestID)

		extractedID := ctx.Value(ContextKeyRequestID).(string)
		assert.Equal(t, requestID, extractedID)
	})

	t.Run("WithAPIKeyID", func(t *testing.T) {
		apiKeyID := uuid.New()
		ctx := WithAPIKeyID(context.Background(), apiKeyID)

		extractedID := ctx.Value(ContextKeyAPIKeyID).(uuid.UUID)
		assert.Equal(t, apiKeyID, extractedID)
	})
}

func TestGenerateRequestID(t *testing.T) {
	t.Run("Generate Unique IDs", func(t *testing.T) {
		id1 := generateRequestID()
		id2 := generateRequestID()
		
		assert.NotEqual(t, id1, id2)
		assert.Contains(t, id1, "req_")
		assert.Contains(t, id2, "req_")
		assert.True(t, len(id1) > 4) // More than just "req_"
		assert.True(t, len(id2) > 4)
	})
}