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
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes/scheme"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// TestSessionTypeValidation tests session type field validation
func TestSessionTypeValidation_ClaudeCodeDefault(t *testing.T) {
	// Setup
	setupTestHandlers()

	// Create request without type field (should default to claude-code)
	req := types.CreateAgenticSessionRequest{
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

	// Verify - should succeed with default type
	if w.Code != http.StatusCreated && w.Code != http.StatusOK {
		t.Errorf("Expected status 200/201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSessionTypeValidation_LangFlowMissingFlowId(t *testing.T) {
	// Setup
	setupTestHandlers()

	// Create langflow request without flowId (should fail)
	req := types.CreateAgenticSessionRequest{
		Type:        "langflow",
		Prompt:      "Test prompt",
		DisplayName: "Test session",
		// FlowID missing - this should fail
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

func TestSessionTypeValidation_LangFlowWithValidFlowId(t *testing.T) {
	// Setup
	setupTestHandlers()

	// Mock LangFlow server that returns a valid flow
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/flows/test-flow-123" {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(langflow.Flow{
				ID:          "test-flow-123",
				Name:        "Test Flow",
				Description: "A test flow",
			})
		}
	}))
	defer mockServer.Close()

	// Set up LangFlowClient with mock server
	handlers.LangFlowClient = langflow.NewClient(mockServer.URL, "test-key")

	// Create langflow request with valid flowId
	req := types.CreateAgenticSessionRequest{
		Type:        "langflow",
		FlowID:      "test-flow-123",
		Prompt:      "Test prompt",
		DisplayName: "Test session",
		FlowInput: map[string]interface{}{
			"message": "test",
		},
	}
	body, _ := json.Marshal(req)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/api/projects/test/agentic-sessions", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("project", "test")

	// Execute
	handlers.CreateSession(c)

	// Verify - should succeed
	if w.Code != http.StatusCreated && w.Code != http.StatusOK {
		t.Errorf("Expected status 200/201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSessionTypeValidation_LangFlowWithInvalidFlowId(t *testing.T) {
	// Setup
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

func TestSessionTypeValidation_LangFlowWithoutClient(t *testing.T) {
	// Setup
	setupTestHandlers()

	// Disable LangFlowClient to test warning path
	originalClient := handlers.LangFlowClient
	handlers.LangFlowClient = nil
	defer func() { handlers.LangFlowClient = originalClient }()

	// Create langflow request with flowId
	req := types.CreateAgenticSessionRequest{
		Type:        "langflow",
		FlowID:      "test-flow-123",
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

	// Verify - should succeed with warning logged (but no validation)
	if w.Code != http.StatusCreated && w.Code != http.StatusOK {
		t.Errorf("Expected status 200/201, got %d: %s", w.Code, w.Body.String())
	}
}

// setupTestHandlers initializes handlers for testing
func setupTestHandlers() {
	// Create fake dynamic client
	s := scheme.Scheme
	dynamicClient := fake.NewSimpleDynamicClient(s)

	// Set up handler dependencies
	handlers.DynamicClient = dynamicClient
	handlers.GetAgenticSessionV1Alpha1Resource = func() schema.GroupVersionResource {
		return schema.GroupVersionResource{
			Group:    "vteam.ambient-code",
			Version:  "v1alpha1",
			Resource: "agenticsessions",
		}
	}
	handlers.GetGitHubToken = func(ctx interface{}, k8s interface{}, dyn dynamic.Interface, ns, name string) (string, error) {
		return "fake-token", nil
	}
	handlers.DeriveRepoFolderFromURL = func(url string) string {
		return "test-repo"
	}
}
