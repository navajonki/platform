package langflow_test

import (
	"ambient-code-backend/langflow"
	"net/http"
	"net/http/httptest"
	"testing"
)

// These tests are intentionally designed to fail to verify our test suite catches bugs

func TestIntentional_WrongHTTPMethod(t *testing.T) {
	// This test verifies that ListFlows uses GET, not POST
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET method for ListFlows, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[]`))
	}))
	defer server.Close()

	client := langflow.NewClient(server.URL, "")
	_, err := client.ListFlows()
	if err != nil {
		t.Errorf("Should not error: %v", err)
	}
}

func TestIntentional_WrongContentType(t *testing.T) {
	// This test verifies that POST requests send JSON content-type
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"session_id":"test","outputs":[]}`))
	}))
	defer server.Close()

	client := langflow.NewClient(server.URL, "")
	req := langflow.FlowExecutionRequest{
		Inputs: map[string]interface{}{"test": "value"},
	}
	_, err := client.RunFlow("test-flow", req)
	if err != nil {
		t.Errorf("Should not error: %v", err)
	}
}

func TestIntentional_APIKeyHeader(t *testing.T) {
	// This test verifies that API key is sent in correct header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiKey := r.Header.Get("x-api-key")
		if apiKey != "my-test-key" {
			t.Errorf("Expected x-api-key header with value 'my-test-key', got '%s'", apiKey)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"healthy"}`))
	}))
	defer server.Close()

	client := langflow.NewClient(server.URL, "my-test-key")
	err := client.CheckHealth()
	if err != nil {
		t.Errorf("Should not error: %v", err)
	}
}

func TestIntentional_CorrectURLPath(t *testing.T) {
	// This test verifies exact URL paths
	tests := []struct {
		name         string
		expectedPath string
		testFunc     func(*langflow.Client) error
	}{
		{
			name:         "CheckHealth",
			expectedPath: "/health",
			testFunc: func(c *langflow.Client) error {
				return c.CheckHealth()
			},
		},
		{
			name:         "ListFlows",
			expectedPath: "/api/v1/flows",
			testFunc: func(c *langflow.Client) error {
				_, err := c.ListFlows()
				return err
			},
		},
		{
			name:         "GetFlow",
			expectedPath: "/api/v1/flows/test-id",
			testFunc: func(c *langflow.Client) error {
				_, err := c.GetFlow("test-id")
				return err
			},
		},
		{
			name:         "RunFlow",
			expectedPath: "/api/v1/run/test-id",
			testFunc: func(c *langflow.Client) error {
				_, err := c.RunFlow("test-id", langflow.FlowExecutionRequest{})
				return err
			},
		},
		{
			name:         "DownloadFlow",
			expectedPath: "/api/v1/flows/download/test-id",
			testFunc: func(c *langflow.Client) error {
				_, err := c.DownloadFlow("test-id")
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != tt.expectedPath {
					t.Errorf("Expected path %s, got %s", tt.expectedPath, r.URL.Path)
				}
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"session_id":"test","outputs":[]}`))
			}))
			defer server.Close()

			client := langflow.NewClient(server.URL, "")
			err := tt.testFunc(client)
			if err != nil {
				// Some paths might still error (like GetFlow with not found)
				// but the path check in the handler would have caught wrong paths
			}
		})
	}
}
