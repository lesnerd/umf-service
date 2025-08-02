package utils

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ufm/internal/telemetry/models"
)

// RespondWithSuccess sends a successful JSON response
func RespondWithSuccess(c *gin.Context, data interface{}) {
	response := models.APIResponse{
		Success:   true,
		Data:      data,
		Timestamp: time.Now(),
	}

	c.JSON(http.StatusOK, response)
}

// RespondWithError sends an error JSON response
func RespondWithError(c *gin.Context, statusCode int, message string) {
	response := models.APIResponse{
		Success:   false,
		Error:     message,
		Timestamp: time.Now(),
	}

	c.JSON(statusCode, response)
}
