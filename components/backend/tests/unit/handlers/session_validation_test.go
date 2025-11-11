package handlers_test

import (
	"ambient-code-backend/handlers"
	"ambient-code-backend/langflow"
	"ambient-code-backend/types"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes/scheme"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// setupTestHandlers initializes required handler dependencies
func setupTestHandlers() {
	// Create fake dynamic client (only needed for DynamicClient != nil check)
	s := scheme.Scheme
	handlers.DynamicClient = fake.NewSimpleDynamicClient(s)
	handlers.GetAgenticSessionV1Alpha1Resource = func() schema.GroupVersionResource {
		return schema.GroupVersionResource{
			Group:    "vteam.ambient-code",
			Version:  "v1alpha1",
			Resource: "agenticsessions",
		}
	}
}

// TestSessionTypeValidation_LangFlowMissingFlowId tests that backend returns 400 when flowId is missing
func TestSessionTypeValidation_LangFlowMissingFlowId(t *testing.T) {
	setupTestHandlers()

	// Create langflow request without flowId (should fail validation)
	req := types.CreateAgenticSessionRequest{
		Type:        "langflow",
		Prompt:      "Test prompt",
		DisplayName: "Test session",
		// FlowID missing - this should fail validation
	}
	body, _ := json.Marshal(req)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/api/projects/test/agentic-sessions", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("project", "test")

	// Execute
	handlers.CreateSession(c)

	// Verify - should return 400 Bad Request
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d: %s", w.Code, w.Body.String())
	}

	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)

	if response["error"] == nil {
		t.Error("Expected error message in response")
	}

	errorMsg := response["error"].(string)
	if errorMsg != "flowId required for langflow sessions" {
		t.Errorf("Expected 'flowId required' error, got: %s", errorMsg)
	}
}

func TestSessionTypeValidation_LangFlowWithInvalidFlowId(t *testing.T) {
	setupTestHandlers()

	// Mock LangFlow server that returns 404 for invalid flow
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error": "flow not found"}`))
	}))
	defer mockServer.Close()

	// Set up LangFlowClient with mock server
	handlers.LangFlowClient = langflow.NewClient(mockServer.URL, "test-key")

	// Create langflow request with invalid flowId
	req := types.CreateAgenticSessionRequest{
		Type:        "langflow",
		FlowID:      "invalid-flow-id",
		Prompt:      "Test prompt",
		DisplayName: "Test session",
	}
	body, _ := json.Marshal(req)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/api/projects/test/agentic-sessions", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("project", "test")

	// Execute
	handlers.CreateSession(c)

	// Verify - should return 400 Bad Request
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d: %s", w.Code, w.Body.String())
	}

	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)

	if response["error"] == nil {
		t.Error("Expected error message in response")
	}

	errorMsg := response["error"].(string)
	if errorMsg == "" || errorMsg == "flowId required for langflow sessions" {
		t.Errorf("Expected 'Invalid flowId' error, got: %s", errorMsg)
	}
}

