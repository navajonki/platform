package langflow_test

import (
	"ambient-code-backend/langflow"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// Test network failures and timeouts
func TestCheckHealth_NetworkError(t *testing.T) {
	// Use invalid URL that will cause connection failure
	client := langflow.NewClient("http://localhost:1", "")
	err := client.CheckHealth()
	if err == nil {
		t.Error("Expected error for network failure, got nil")
	}
}

func TestCheckHealth_Timeout(t *testing.T) {
	// Server that never responds
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(35 * time.Second) // Longer than client timeout
	}))
	defer server.Close()

	client := langflow.NewClient(server.URL, "")
	err := client.CheckHealth()
	if err == nil {
		t.Error("Expected timeout error, got nil")
	}
	if !strings.Contains(err.Error(), "timeout") && !strings.Contains(err.Error(), "deadline") {
		t.Errorf("Expected timeout error, got: %v", err)
	}
}

// Test malformed responses
func TestListFlows_MalformedJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{invalid json}`))
	}))
	defer server.Close()

	client := langflow.NewClient(server.URL, "")
	flows, err := client.ListFlows()
	if err == nil {
		t.Error("Expected error for malformed JSON, got nil")
	}
	if flows != nil {
		t.Errorf("Expected nil flows on error, got %v", flows)
	}
}

func TestListFlows_EmptyResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[]`))
	}))
	defer server.Close()

	client := langflow.NewClient(server.URL, "")
	flows, err := client.ListFlows()
	if err != nil {
		t.Errorf("Expected no error for empty array, got %v", err)
	}
	if len(flows) != 0 {
		t.Errorf("Expected 0 flows, got %d", len(flows))
	}
}

func TestListFlows_UnexpectedStatusCode(t *testing.T) {
	statusCodes := []int{
		http.StatusUnauthorized,
		http.StatusForbidden,
		http.StatusInternalServerError,
		http.StatusBadGateway,
	}

	for _, code := range statusCodes {
		t.Run(http.StatusText(code), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(code)
				w.Write([]byte(`{"error":"test error"}`))
			}))
			defer server.Close()

			client := langflow.NewClient(server.URL, "")
			flows, err := client.ListFlows()
			if err == nil {
				t.Errorf("Expected error for status %d, got nil", code)
			}
			if flows != nil {
				t.Error("Expected nil flows on error")
			}
		})
	}
}

// Test GetFlow edge cases
func TestGetFlow_EmptyID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := langflow.NewClient(server.URL, "")
	flow, err := client.GetFlow("")

	// The client should still make the request (API will handle validation)
	// But let's verify it doesn't crash
	if err == nil {
		t.Error("Expected error for empty ID")
	}
	if flow != nil {
		t.Error("Expected nil flow for empty ID")
	}
}

func TestGetFlow_SpecialCharacters(t *testing.T) {
	specialIDs := []string{
		"flow-with-spaces in it",
		"flow/with/slashes",
		"flow?with=query",
		"flow#with#hash",
	}

	for _, id := range specialIDs {
		t.Run(id, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Just verify we can handle the request
				w.WriteHeader(http.StatusNotFound)
			}))
			defer server.Close()

			client := langflow.NewClient(server.URL, "")
			flow, err := client.GetFlow(id)

			// Should get error, but shouldn't crash
			if err == nil {
				t.Error("Expected error for special character ID")
			}
			if flow != nil {
				t.Error("Expected nil flow")
			}
		})
	}
}

// Test RunFlow edge cases
func TestRunFlow_EmptyRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req langflow.FlowExecutionRequest
		json.NewDecoder(r.Body).Decode(&req)

		// Should set default output_type even with empty request
		if req.OutputType != "chat" {
			t.Errorf("Expected default output_type 'chat', got '%s'", req.OutputType)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(langflow.FlowExecutionResponse{})
	}))
	defer server.Close()

	client := langflow.NewClient(server.URL, "")
	req := langflow.FlowExecutionRequest{} // Completely empty

	_, err := client.RunFlow("test-flow", req)
	if err != nil {
		t.Errorf("Expected to handle empty request, got error: %v", err)
	}
}

func TestRunFlow_LargePayload(t *testing.T) {
	// Create large input to test payload handling
	largeInputs := make(map[string]interface{})
	for i := 0; i < 1000; i++ {
		largeInputs[string(rune(i))] = strings.Repeat("x", 1000)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(langflow.FlowExecutionResponse{
			SessionID: "test-session",
		})
	}))
	defer server.Close()

	client := langflow.NewClient(server.URL, "")
	req := langflow.FlowExecutionRequest{
		Inputs: largeInputs,
	}

	resp, err := client.RunFlow("test-flow", req)
	if err != nil {
		t.Errorf("Expected to handle large payload, got error: %v", err)
	}
	if resp == nil {
		t.Error("Expected response for large payload")
	}
}

func TestRunFlow_NilInputs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req langflow.FlowExecutionRequest
		json.NewDecoder(r.Body).Decode(&req)

		// Nil inputs should be handled
		if req.Inputs == nil {
			req.Inputs = make(map[string]interface{})
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(langflow.FlowExecutionResponse{})
	}))
	defer server.Close()

	client := langflow.NewClient(server.URL, "")
	req := langflow.FlowExecutionRequest{
		Inputs: nil, // Explicitly nil
	}

	_, err := client.RunFlow("test-flow", req)
	if err != nil {
		t.Errorf("Expected to handle nil inputs, got error: %v", err)
	}
}

// Test DownloadFlow edge cases
func TestDownloadFlow_EmptyResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		// Empty response body
	}))
	defer server.Close()

	client := langflow.NewClient(server.URL, "")
	data, err := client.DownloadFlow("test-flow")
	if err != nil {
		t.Errorf("Expected to handle empty response, got error: %v", err)
	}
	if len(data) != 0 {
		t.Errorf("Expected empty data, got %d bytes", len(data))
	}
}

func TestDownloadFlow_BinaryData(t *testing.T) {
	// Test with non-JSON binary data
	binaryData := []byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0xFD}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write(binaryData)
	}))
	defer server.Close()

	client := langflow.NewClient(server.URL, "")
	data, err := client.DownloadFlow("test-flow")
	if err != nil {
		t.Errorf("Expected to handle binary data, got error: %v", err)
	}
	if len(data) != len(binaryData) {
		t.Errorf("Expected %d bytes, got %d", len(binaryData), len(data))
	}
}

// Test client initialization edge cases
func TestNewClient_EmptyBaseURL(t *testing.T) {
	client := langflow.NewClient("", "test-key")
	if client == nil {
		t.Fatal("Expected client to be created even with empty URL")
	}

	// Should fail when trying to use it
	err := client.CheckHealth()
	if err == nil {
		t.Error("Expected error when using client with empty URL")
	}
}

func TestNewClient_InvalidURL(t *testing.T) {
	invalidURLs := []string{
		"not-a-url",
		"://missing-scheme",
		"http://",
		"ftp://invalid-scheme",
	}

	for _, url := range invalidURLs {
		t.Run(url, func(t *testing.T) {
			client := langflow.NewClient(url, "")
			if client == nil {
				t.Fatal("Expected client to be created")
			}

			// Should fail when trying to use
			err := client.CheckHealth()
			if err == nil {
				t.Errorf("Expected error for invalid URL: %s", url)
			}
		})
	}
}

// Test concurrent requests
func TestClient_ConcurrentRequests(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		time.Sleep(10 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]langflow.Flow{})
	}))
	defer server.Close()

	client := langflow.NewClient(server.URL, "")

	// Make concurrent requests
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			_, err := client.ListFlows()
			if err != nil {
				t.Errorf("Concurrent request failed: %v", err)
			}
			done <- true
		}()
	}

	// Wait for all to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	if requestCount != 10 {
		t.Errorf("Expected 10 requests, got %d", requestCount)
	}
}
