package utils

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/ufm/internal/telemetry/models"
)

func setupTestContext() (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	return c, w
}

func TestRespondWithSuccess(t *testing.T) {
	c, w := setupTestContext()

	// Test data
	testData := map[string]interface{}{
		"message": "test success",
		"count":   42,
	}

	// Call the function
	RespondWithSuccess(c, testData)

	// Assert response
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))

	// Parse response
	var response models.APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	// Assert response fields
	assert.True(t, response.Success)
	// JSON unmarshaling converts numbers to float64, so we need to check individual fields
	responseData := response.Data.(map[string]interface{})
	assert.Equal(t, "test success", responseData["message"])
	assert.Equal(t, float64(42), responseData["count"])
	assert.Empty(t, response.Error)
	assert.False(t, response.Timestamp.IsZero())
}

func TestRespondWithSuccess_EmptyData(t *testing.T) {
	c, w := setupTestContext()

	// Call with nil data
	RespondWithSuccess(c, nil)

	// Assert response
	assert.Equal(t, http.StatusOK, w.Code)

	var response models.APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.True(t, response.Success)
	assert.Nil(t, response.Data)
	assert.Empty(t, response.Error)
}

func TestRespondWithSuccess_ComplexData(t *testing.T) {
	c, w := setupTestContext()

	// Test with complex nested data
	testData := map[string]interface{}{
		"user": map[string]interface{}{
			"id":   123,
			"name": "John Doe",
			"tags": []string{"admin", "user"},
		},
		"metadata": map[string]interface{}{
			"created_at": time.Now().Format(time.RFC3339),
			"version":    "1.0.0",
		},
	}

	RespondWithSuccess(c, testData)

	assert.Equal(t, http.StatusOK, w.Code)

	var response models.APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.True(t, response.Success)
	// JSON unmarshaling converts numbers to float64, so we need to check individual fields
	responseData := response.Data.(map[string]interface{})
	userData := responseData["user"].(map[string]interface{})
	metadataData := responseData["metadata"].(map[string]interface{})

	assert.Equal(t, float64(123), userData["id"])
	assert.Equal(t, "John Doe", userData["name"])
	assert.Equal(t, []interface{}{"admin", "user"}, userData["tags"])
	assert.Contains(t, metadataData, "created_at")
	assert.Equal(t, "1.0.0", metadataData["version"])
}

func TestRespondWithError(t *testing.T) {
	c, w := setupTestContext()

	// Test error response
	errorMessage := "Something went wrong"
	statusCode := http.StatusInternalServerError

	RespondWithError(c, statusCode, errorMessage)

	// Assert response
	assert.Equal(t, statusCode, w.Code)
	assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))

	// Parse response
	var response models.APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	// Assert response fields
	assert.False(t, response.Success)
	assert.Equal(t, errorMessage, response.Error)
	assert.Nil(t, response.Data)
	assert.False(t, response.Timestamp.IsZero())
}

func TestRespondWithError_DifferentStatusCodes(t *testing.T) {
	testCases := []struct {
		name       string
		statusCode int
		message    string
	}{
		{
			name:       "bad request",
			statusCode: http.StatusBadRequest,
			message:    "Invalid request",
		},
		{
			name:       "not found",
			statusCode: http.StatusNotFound,
			message:    "Resource not found",
		},
		{
			name:       "unauthorized",
			statusCode: http.StatusUnauthorized,
			message:    "Authentication required",
		},
		{
			name:       "forbidden",
			statusCode: http.StatusForbidden,
			message:    "Access denied",
		},
		{
			name:       "internal server error",
			statusCode: http.StatusInternalServerError,
			message:    "Internal server error",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			c, w := setupTestContext()

			RespondWithError(c, tc.statusCode, tc.message)

			assert.Equal(t, tc.statusCode, w.Code)

			var response models.APIResponse
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)

			assert.False(t, response.Success)
			assert.Equal(t, tc.message, response.Error)
			assert.Nil(t, response.Data)
		})
	}
}

func TestRespondWithError_EmptyMessage(t *testing.T) {
	c, w := setupTestContext()

	RespondWithError(c, http.StatusBadRequest, "")

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response models.APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.False(t, response.Success)
	assert.Empty(t, response.Error)
}

func TestRespondWithError_LongMessage(t *testing.T) {
	c, w := setupTestContext()

	// Test with a very long error message
	longMessage := "This is a very long error message that contains a lot of details about what went wrong. " +
		"It might include stack traces, error codes, and other debugging information that could be useful " +
		"for developers to understand and fix the issue. The message should be properly handled and not " +
		"cause any issues with the JSON serialization or HTTP response."

	RespondWithError(c, http.StatusInternalServerError, longMessage)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response models.APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.False(t, response.Success)
	assert.Equal(t, longMessage, response.Error)
}

func TestResponseTimestamp(t *testing.T) {
	c, w := setupTestContext()

	// Record time before call
	beforeTime := time.Now()

	RespondWithSuccess(c, "test")

	// Record time after call
	afterTime := time.Now()

	var response models.APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	// Verify timestamp is within expected range
	assert.True(t, response.Timestamp.After(beforeTime) || response.Timestamp.Equal(beforeTime))
	assert.True(t, response.Timestamp.Before(afterTime) || response.Timestamp.Equal(afterTime))
}

func TestResponseJSONStructure(t *testing.T) {
	c, w := setupTestContext()

	testData := map[string]string{
		"key1": "value1",
		"key2": "value2",
	}

	RespondWithSuccess(c, testData)

	// Verify JSON is valid
	var response models.APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	// Verify all expected fields are present
	assert.True(t, response.Success)
	assert.NotNil(t, response.Data)
	assert.False(t, response.Timestamp.IsZero())
	assert.Empty(t, response.Error) // Should be empty in success response
}

func TestErrorResponseJSONStructure(t *testing.T) {
	c, w := setupTestContext()

	RespondWithError(c, http.StatusBadRequest, "test error")

	var response models.APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	// Verify all expected fields are present
	assert.False(t, response.Success)
	assert.NotEmpty(t, response.Error)
	assert.False(t, response.Timestamp.IsZero())
	assert.Nil(t, response.Data) // Should be nil in error response
}

func TestResponseContentType(t *testing.T) {
	testCases := []struct {
		name     string
		function func(*gin.Context)
	}{
		{
			name: "success response",
			function: func(c *gin.Context) {
				RespondWithSuccess(c, "test")
			},
		},
		{
			name: "error response",
			function: func(c *gin.Context) {
				RespondWithError(c, http.StatusBadRequest, "test error")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			c, w := setupTestContext()

			tc.function(c)

			contentType := w.Header().Get("Content-Type")
			assert.Equal(t, "application/json; charset=utf-8", contentType)
		})
	}
}
