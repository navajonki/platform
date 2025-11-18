package langflow

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client handles communication with LangFlow API
type Client struct {
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client
}

// Flow represents a LangFlow workflow
type Flow struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Data        map[string]interface{} `json:"data"`
	UpdatedAt   string                 `json:"updated_at"`
}

// FlowsResponse represents the response from GET /api/v1/flows
type FlowsResponse struct {
	Flows []Flow `json:"flows"`
}

// FlowExecutionRequest represents a request to execute a flow
type FlowExecutionRequest struct {
	InputValue string                 `json:"input_value,omitempty"`
	Tweaks     map[string]interface{} `json:"tweaks,omitempty"`
	Inputs     map[string]interface{} `json:"inputs,omitempty"`
	OutputType string                 `json:"output_type,omitempty"`
}

// FlowExecutionResponse represents the response from flow execution
type FlowExecutionResponse struct {
	SessionID string       `json:"session_id"`
	Outputs   []FlowOutput `json:"outputs"`
}

// FlowOutput represents a single output from flow execution
type FlowOutput struct {
	Type    string                 `json:"type"`
	Message string                 `json:"message,omitempty"`
	Data    map[string]interface{} `json:"data"`
}

// NewClient creates a new LangFlow API client
func NewClient(baseURL, apiKey string) *Client {
	return &Client{
		BaseURL: baseURL,
		APIKey:  apiKey,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// doRequest performs an HTTP request with authentication
func (c *Client) doRequest(method, path string, body interface{}) (*http.Response, error) {
	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonData)
	}

	url := fmt.Sprintf("%s%s", c.BaseURL, path)
	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Encoding", "gzip, deflate")
	if c.APIKey != "" {
		req.Header.Set("x-api-key", c.APIKey)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	// Handle gzip-encoded responses
	if strings.Contains(resp.Header.Get("Content-Encoding"), "gzip") {
		gzReader, err := gzip.NewReader(resp.Body)
		if err != nil {
			resp.Body.Close()
			return nil, fmt.Errorf("failed to create gzip reader: %w", err)
		}
		// Replace the response body with the decompressed version
		resp.Body = io.NopCloser(io.TeeReader(gzReader, io.Discard))
		// Create a new response with decompressed body
		decompressed, err := io.ReadAll(gzReader)
		if err != nil {
			gzReader.Close()
			resp.Body.Close()
			return nil, fmt.Errorf("failed to decompress response: %w", err)
		}
		gzReader.Close()
		resp.Body = io.NopCloser(bytes.NewReader(decompressed))
	}

	return resp, nil
}

// CheckHealth checks if LangFlow is healthy and accessible
func (c *Client) CheckHealth() error {
	resp, err := c.doRequest("GET", "/health", nil)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check returned status %d", resp.StatusCode)
	}

	return nil
}

// ListFlows retrieves all available flows from LangFlow
func (c *Client) ListFlows() ([]Flow, error) {
	resp, err := c.doRequest("GET", "/api/v1/flows/", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list flows: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("list flows returned status %d: %s", resp.StatusCode, string(body))
	}

	// LangFlow API returns different structures, handle both
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Try parsing as array first (direct flow list)
	var flows []Flow
	if err := json.Unmarshal(bodyBytes, &flows); err == nil {
		return flows, nil
	}

	// Try parsing as object with flows field
	var flowsResp FlowsResponse
	if err := json.Unmarshal(bodyBytes, &flowsResp); err != nil {
		return nil, fmt.Errorf("failed to parse flows response: %w", err)
	}

	return flowsResp.Flows, nil
}

// GetFlow retrieves a specific flow by ID
func (c *Client) GetFlow(id string) (*Flow, error) {
	path := fmt.Sprintf("/api/v1/flows/%s", id)
	resp, err := c.doRequest("GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get flow: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("flow not found")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("get flow returned status %d: %s", resp.StatusCode, string(body))
	}

	var flow Flow
	if err := json.NewDecoder(resp.Body).Decode(&flow); err != nil {
		return nil, fmt.Errorf("failed to decode flow: %w", err)
	}

	return &flow, nil
}

// RunFlow executes a flow with the given inputs
func (c *Client) RunFlow(id string, req FlowExecutionRequest) (*FlowExecutionResponse, error) {
	// Set default output type if not specified
	if req.OutputType == "" {
		req.OutputType = "chat"
	}

	path := fmt.Sprintf("/api/v1/run/%s", id)
	resp, err := c.doRequest("POST", path, req)
	if err != nil {
		return nil, fmt.Errorf("failed to run flow: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("run flow returned status %d: %s", resp.StatusCode, string(body))
	}

	var result FlowExecutionResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode execution response: %w", err)
	}

	return &result, nil
}

// DownloadFlow downloads the flow definition as JSON
func (c *Client) DownloadFlow(id string) ([]byte, error) {
	path := fmt.Sprintf("/api/v1/flows/download/%s", id)
	resp, err := c.doRequest("GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to download flow: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("download flow returned status %d: %s", resp.StatusCode, string(body))
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read flow data: %w", err)
	}

	return data, nil
}
