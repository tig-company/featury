package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func SetupRoutes(r *gin.Engine) {
	r.GET("/health", healthCheck)
	
	v1 := r.Group("/api/v1")
	{
		features := v1.Group("/features")
		{
			features.GET("", getFeatures)
			features.POST("", createFeature)
			features.GET("/:id", getFeature)
			features.PUT("/:id", updateFeature)
			features.DELETE("/:id", deleteFeature)
		}
	}
}

func healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"service": "featury",
		"version": "1.0.0",
	})
}

func getFeatures(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"features": []interface{}{},
		"total":    0,
	})
}

func createFeature(c *gin.Context) {
	c.JSON(http.StatusCreated, gin.H{
		"message": "Feature created successfully",
	})
}

func getFeature(c *gin.Context) {
	id := c.Param("id")
	c.JSON(http.StatusOK, gin.H{
		"id":      id,
		"message": "Feature retrieved successfully",
	})
}

func updateFeature(c *gin.Context) {
	id := c.Param("id")
	c.JSON(http.StatusOK, gin.H{
		"id":      id,
		"message": "Feature updated successfully",
	})
}

func deleteFeature(c *gin.Context) {
	id := c.Param("id")
	c.JSON(http.StatusOK, gin.H{
		"id":      id,
		"message": "Feature deleted successfully",
	})
}