package langflow_test

import (
	"ambient-code-backend/langflow"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name    string
		baseURL string
		apiKey  string
	}{
		{
			name:    "with API key",
			baseURL: "http://langflow.test",
			apiKey:  "test-key",
		},
		{
			name:    "without API key",
			baseURL: "http://langflow.test",
			apiKey:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := langflow.NewClient(tt.baseURL, tt.apiKey)
			if client == nil {
				t.Fatal("Expected client to be non-nil")
			}
			if client.BaseURL != tt.baseURL {
				t.Errorf("Expected BaseURL %s, got %s", tt.baseURL, client.BaseURL)
			}
			if client.APIKey != tt.apiKey {
				t.Errorf("Expected APIKey %s, got %s", tt.apiKey, client.APIKey)
			}
		})
	}
}

func TestCheckHealth_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/health" {
			t.Errorf("Expected path /health, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"healthy"}`))
	}))
	defer server.Close()

	client := langflow.NewClient(server.URL, "")
	err := client.CheckHealth()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestCheckHealth_Failure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	client := langflow.NewClient(server.URL, "")
	err := client.CheckHealth()
	if err == nil {
		t.Error("Expected error for unhealthy service, got nil")
	}
}

func TestListFlows_Success(t *testing.T) {
	expectedFlows := []langflow.Flow{
		{
			ID:          "flow-1",
			Name:        "Test Flow 1",
			Description: "Description 1",
			UpdatedAt:   "2025-01-10T12:00:00Z",
		},
		{
			ID:          "flow-2",
			Name:        "Test Flow 2",
			Description: "Description 2",
			UpdatedAt:   "2025-01-10T13:00:00Z",
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/flows" {
			t.Errorf("Expected path /api/v1/flows, got %s", r.URL.Path)
		}

		// Check API key header if present
		if apiKey := r.Header.Get("x-api-key"); apiKey != "" && apiKey != "test-key" {
			t.Errorf("Expected API key test-key, got %s", apiKey)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(expectedFlows)
	}))
	defer server.Close()

	client := langflow.NewClient(server.URL, "test-key")
	flows, err := client.ListFlows()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(flows) != len(expectedFlows) {
		t.Errorf("Expected %d flows, got %d", len(expectedFlows), len(flows))
	}

	for i, flow := range flows {
		if flow.ID != expectedFlows[i].ID {
			t.Errorf("Flow %d: Expected ID %s, got %s", i, expectedFlows[i].ID, flow.ID)
		}
		if flow.Name != expectedFlows[i].Name {
			t.Errorf("Flow %d: Expected Name %s, got %s", i, expectedFlows[i].Name, flow.Name)
		}
	}
}

func TestListFlows_WithFlowsWrapper(t *testing.T) {
	// Test alternate response format with "flows" wrapper
	expectedFlows := []langflow.Flow{
		{
			ID:   "flow-1",
			Name: "Test Flow",
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"flows": expectedFlows,
		})
	}))
	defer server.Close()

	client := langflow.NewClient(server.URL, "")
	flows, err := client.ListFlows()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(flows) != 1 {
		t.Errorf("Expected 1 flow, got %d", len(flows))
	}
}

func TestGetFlow_Success(t *testing.T) {
	expectedFlow := langflow.Flow{
		ID:          "flow-123",
		Name:        "Test Flow",
		Description: "A test flow",
		UpdatedAt:   "2025-01-10T12:00:00Z",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPath := "/api/v1/flows/flow-123"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(expectedFlow)
	}))
	defer server.Close()

	client := langflow.NewClient(server.URL, "")
	flow, err := client.GetFlow("flow-123")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if flow.ID != expectedFlow.ID {
		t.Errorf("Expected ID %s, got %s", expectedFlow.ID, flow.ID)
	}
	if flow.Name != expectedFlow.Name {
		t.Errorf("Expected Name %s, got %s", expectedFlow.Name, flow.Name)
	}
}

func TestGetFlow_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":"flow not found"}`))
	}))
	defer server.Close()

	client := langflow.NewClient(server.URL, "")
	flow, err := client.GetFlow("nonexistent")
	if err == nil {
		t.Error("Expected error for not found flow, got nil")
	}
	if flow != nil {
		t.Error("Expected nil flow for not found, got non-nil")
	}
}

func TestRunFlow_Success(t *testing.T) {
	expectedResponse := langflow.FlowExecutionResponse{
		SessionID: "session-123",
		Outputs: []langflow.FlowOutput{
			{
				Type:    "chat",
				Message: "Test output",
				Data:    map[string]interface{}{"result": "success"},
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST method, got %s", r.Method)
		}

		expectedPath := "/api/v1/run/flow-123"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
		}

		// Verify request body
		var req langflow.FlowExecutionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("Failed to decode request body: %v", err)
		}

		if req.OutputType != "chat" {
			t.Errorf("Expected output_type chat, got %s", req.OutputType)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(expectedResponse)
	}))
	defer server.Close()

	client := langflow.NewClient(server.URL, "test-key")
	req := langflow.FlowExecutionRequest{
		Inputs: map[string]interface{}{
			"message": "test input",
		},
	}

	resp, err := client.RunFlow("flow-123", req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if resp.SessionID != expectedResponse.SessionID {
		t.Errorf("Expected SessionID %s, got %s", expectedResponse.SessionID, resp.SessionID)
	}

	if len(resp.Outputs) != len(expectedResponse.Outputs) {
		t.Errorf("Expected %d outputs, got %d", len(expectedResponse.Outputs), len(resp.Outputs))
	}
}

func TestRunFlow_DefaultOutputType(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req langflow.FlowExecutionRequest
		json.NewDecoder(r.Body).Decode(&req)

		// Verify default output_type is set to "chat"
		if req.OutputType != "chat" {
			t.Errorf("Expected default output_type 'chat', got '%s'", req.OutputType)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(langflow.FlowExecutionResponse{})
	}))
	defer server.Close()

	client := langflow.NewClient(server.URL, "")
	req := langflow.FlowExecutionRequest{
		Inputs: map[string]interface{}{"test": "value"},
	}

	_, err := client.RunFlow("flow-123", req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestRunFlow_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"invalid input"}`))
	}))
	defer server.Close()

	client := langflow.NewClient(server.URL, "")
	req := langflow.FlowExecutionRequest{}

	resp, err := client.RunFlow("flow-123", req)
	if err == nil {
		t.Error("Expected error for bad request, got nil")
	}
	if resp != nil {
		t.Error("Expected nil response for error, got non-nil")
	}
}

func TestDownloadFlow_Success(t *testing.T) {
	expectedData := []byte(`{"nodes": [], "edges": []}`)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPath := "/api/v1/flows/download/flow-123"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
		w.Write(expectedData)
	}))
	defer server.Close()

	client := langflow.NewClient(server.URL, "")
	data, err := client.DownloadFlow("flow-123")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if string(data) != string(expectedData) {
		t.Errorf("Expected data %s, got %s", expectedData, data)
	}
}
