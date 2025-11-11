package langflow_test

import (
	"ambient-code-backend/handlers"
	"ambient-code-backend/langflow"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestLangFlowHealth_ClientNotConfigured(t *testing.T) {
	// Save and restore original client
	originalClient := handlers.LangFlowClient
	defer func() { handlers.LangFlowClient = originalClient }()

	handlers.LangFlowClient = nil

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/langflow/health", nil)

	handlers.LangFlowHealth(c)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status %d, got %d", http.StatusServiceUnavailable, w.Code)
	}

	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)

	if response["available"] != false {
		t.Error("Expected available to be false")
	}

	if response["error"] == nil {
		t.Error("Expected error message in response")
	}
}

func TestLangFlowHealth_ServiceHealthy(t *testing.T) {
	// Mock LangFlow server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"healthy"}`))
		}
	}))
	defer mockServer.Close()

	// Save and restore original client
	originalClient := handlers.LangFlowClient
	defer func() { handlers.LangFlowClient = originalClient }()

	handlers.LangFlowClient = langflow.NewClient(mockServer.URL, "")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/langflow/health", nil)

	handlers.LangFlowHealth(c)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)

	if response["available"] != true {
		t.Error("Expected available to be true")
	}

	if response["status"] != "healthy" {
		t.Errorf("Expected status 'healthy', got %v", response["status"])
	}
}

func TestLangFlowHealth_ServiceUnhealthy(t *testing.T) {
	// Mock LangFlow server that's unhealthy
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
	}))
	defer mockServer.Close()

	// Save and restore original client
	originalClient := handlers.LangFlowClient
	defer func() { handlers.LangFlowClient = originalClient }()

	handlers.LangFlowClient = langflow.NewClient(mockServer.URL, "")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/langflow/health", nil)

	handlers.LangFlowHealth(c)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status %d, got %d", http.StatusServiceUnavailable, w.Code)
	}

	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)

	if response["available"] != false {
		t.Error("Expected available to be false")
	}
}

func TestListLangFlowFlows_Success(t *testing.T) {
	expectedFlows := []langflow.Flow{
		{
			ID:          "flow-1",
			Name:        "Test Flow 1",
			Description: "Description 1",
		},
		{
			ID:          "flow-2",
			Name:        "Test Flow 2",
			Description: "Description 2",
		},
	}

	// Mock LangFlow server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/flows" {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(expectedFlows)
		}
	}))
	defer mockServer.Close()

	// Save and restore original client
	originalClient := handlers.LangFlowClient
	defer func() { handlers.LangFlowClient = originalClient }()

	handlers.LangFlowClient = langflow.NewClient(mockServer.URL, "test-key")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/langflow/flows", nil)

	handlers.ListLangFlowFlows(c)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)

	flows, ok := response["flows"].([]interface{})
	if !ok {
		t.Fatal("Expected flows array in response")
	}

	if len(flows) != len(expectedFlows) {
		t.Errorf("Expected %d flows, got %d", len(expectedFlows), len(flows))
	}
}

func TestListLangFlowFlows_ClientNotConfigured(t *testing.T) {
	// Save and restore original client
	originalClient := handlers.LangFlowClient
	defer func() { handlers.LangFlowClient = originalClient }()

	handlers.LangFlowClient = nil

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/langflow/flows", nil)

	handlers.ListLangFlowFlows(c)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status %d, got %d", http.StatusServiceUnavailable, w.Code)
	}
}

func TestGetLangFlowFlow_Success(t *testing.T) {
	expectedFlow := langflow.Flow{
		ID:          "flow-123",
		Name:        "Test Flow",
		Description: "A test flow",
	}

	// Mock LangFlow server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/flows/flow-123" {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(expectedFlow)
		}
	}))
	defer mockServer.Close()

	// Save and restore original client
	originalClient := handlers.LangFlowClient
	defer func() { handlers.LangFlowClient = originalClient }()

	handlers.LangFlowClient = langflow.NewClient(mockServer.URL, "")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/langflow/flows/flow-123", nil)
	c.Params = gin.Params{{Key: "id", Value: "flow-123"}}

	handlers.GetLangFlowFlow(c)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var flow langflow.Flow
	json.NewDecoder(w.Body).Decode(&flow)

	if flow.ID != expectedFlow.ID {
		t.Errorf("Expected ID %s, got %s", expectedFlow.ID, flow.ID)
	}
	if flow.Name != expectedFlow.Name {
		t.Errorf("Expected Name %s, got %s", expectedFlow.Name, flow.Name)
	}
}

func TestGetLangFlowFlow_NotFound(t *testing.T) {
	// Mock LangFlow server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":"flow not found"}`))
	}))
	defer mockServer.Close()

	// Save and restore original client
	originalClient := handlers.LangFlowClient
	defer func() { handlers.LangFlowClient = originalClient }()

	handlers.LangFlowClient = langflow.NewClient(mockServer.URL, "")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/langflow/flows/nonexistent", nil)
	c.Params = gin.Params{{Key: "id", Value: "nonexistent"}}

	handlers.GetLangFlowFlow(c)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, w.Code)
	}

	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)

	if response["error"] == nil {
		t.Error("Expected error message in response")
	}
}

func TestGetLangFlowFlow_MissingID(t *testing.T) {
	// Save and restore original client
	originalClient := handlers.LangFlowClient
	defer func() { handlers.LangFlowClient = originalClient }()

	handlers.LangFlowClient = langflow.NewClient("http://test", "")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/langflow/flows/", nil)
	c.Params = gin.Params{{Key: "id", Value: ""}}

	handlers.GetLangFlowFlow(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}
