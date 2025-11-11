package handlers

import (
	"ambient-code-backend/langflow"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// LangFlowClient is the global LangFlow client instance
// Set by main.go during initialization
var LangFlowClient *langflow.Client

// LangFlowHealth checks if LangFlow service is accessible
func LangFlowHealth(c *gin.Context) {
	if LangFlowClient == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"available": false,
			"error":     "LangFlow client not configured",
		})
		return
	}

	err := LangFlowClient.CheckHealth()
	if err != nil {
		log.Printf("LangFlow health check failed: %v", err)
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"available": false,
			"error":     err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"available": true,
		"status":    "healthy",
	})
}

// ListLangFlowFlows retrieves all available LangFlow flows
func ListLangFlowFlows(c *gin.Context) {
	if LangFlowClient == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "LangFlow client not configured",
		})
		return
	}

	flows, err := LangFlowClient.ListFlows()
	if err != nil {
		log.Printf("Failed to list LangFlow flows: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch flows from LangFlow",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"flows": flows,
	})
}

// GetLangFlowFlow retrieves a specific flow by ID
func GetLangFlowFlow(c *gin.Context) {
	flowID := c.Param("id")
	if flowID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Flow ID is required",
		})
		return
	}

	if LangFlowClient == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "LangFlow client not configured",
		})
		return
	}

	flow, err := LangFlowClient.GetFlow(flowID)
	if err != nil {
		log.Printf("Failed to get LangFlow flow %s: %v", flowID, err)
		if err.Error() == "flow not found" {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Flow not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch flow from LangFlow",
		})
		return
	}

	c.JSON(http.StatusOK, flow)
}
