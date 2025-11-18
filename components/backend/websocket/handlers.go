package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"ambient-code-backend/handlers"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// WebSocket upgrader
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// Allow all origins for development - should be restricted in production
		return true
	},
}

// HandleSessionWebSocket handles WebSocket connections for sessions
// Route: /projects/:projectName/sessions/:sessionId/ws
func HandleSessionWebSocket(c *gin.Context) {
	sessionID := c.Param("sessionId")
	log.Printf("handleSessionWebSocket for session: %s", sessionID)

	// Access enforced by RBAC on downstream resources

	// Best-effort user identity: prefer forwarded user, else extract ServiceAccount from bearer token
	var userIDStr string
	if v, ok := c.Get("userID"); ok {
		if s, ok2 := v.(string); ok2 {
			userIDStr = s
		}
	}
	if userIDStr == "" {
		if ns, sa, ok := handlers.ExtractServiceAccountFromAuth(c); ok {
			userIDStr = ns + ":" + sa
		}
	}

	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	sessionConn := &SessionConnection{
		SessionID: sessionID,
		Conn:      conn,
		UserID:    userIDStr,
	}

	// Register connection
	Hub.register <- sessionConn

	// Handle messages from client
	go handleWebSocketMessages(sessionConn)

	// Keep connection alive
	go handleWebSocketPing(sessionConn)
}

// handleWebSocketMessages processes incoming WebSocket messages
func handleWebSocketMessages(conn *SessionConnection) {
	defer func() {
		Hub.unregister <- conn
	}()

	for {
		messageType, messageData, err := conn.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		if messageType == websocket.TextMessage {
			var msg map[string]interface{}
			if err := json.Unmarshal(messageData, &msg); err != nil {
				log.Printf("Failed to parse WebSocket message: %v", err)
				continue
			}

			// Handle control messages
			if msgType, ok := msg["type"].(string); ok {
				if msgType == "ping" {
					// Respond with pong
					pong := map[string]interface{}{
						"type":      "pong",
						"timestamp": time.Now().UTC().Format(time.RFC3339),
					}
					pongData, _ := json.Marshal(pong)
					// Lock write mutex before writing pong
					conn.writeMu.Lock()
					_ = conn.Conn.WriteMessage(websocket.TextMessage, pongData)
					conn.writeMu.Unlock()
					continue
				}
				// Extract payload from runner message to avoid double-nesting
				// Runner sends: {type, seq, timestamp, payload}
				// We only want to store the payload field
				payload, ok := msg["payload"].(map[string]interface{})
				if !ok {
					payload = msg // Fallback for legacy format
				}
				// Broadcast all other messages to session listeners (UI and others)
				sessionMsg := &SessionMessage{
					SessionID: conn.SessionID,
					Type:      msgType,
					Timestamp: time.Now().UTC().Format(time.RFC3339),
					Payload:   payload,
				}
				Hub.broadcast <- sessionMsg
			}
		}
	}
}

// handleWebSocketPing sends periodic ping messages
func handleWebSocketPing(conn *SessionConnection) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		// Lock write mutex before writing ping
		conn.writeMu.Lock()
		err := conn.Conn.WriteMessage(websocket.PingMessage, nil)
		conn.writeMu.Unlock()
		if err != nil {
			return
		}
	}
}

// GetSessionMessagesWS handles GET /projects/:projectName/sessions/:sessionId/messages
// Retrieves messages from S3 storage for Claude Code sessions or CR status for LangFlow sessions
func GetSessionMessagesWS(c *gin.Context) {
	sessionID := c.Param("sessionId")
	projectName := c.Param("projectName")

	// Access enforced by RBAC on downstream resources

	// Retrieve session to determine type
	var messages []SessionMessage
	var err error

	// Try to get session type from Kubernetes
	if handlers.DynamicClient != nil && handlers.GetAgenticSessionV1Alpha1Resource != nil {
		gvr := handlers.GetAgenticSessionV1Alpha1Resource()
		ctx := context.Background()

		sessionObj, getErr := handlers.DynamicClient.Resource(gvr).Namespace(projectName).Get(ctx, sessionID, v1.GetOptions{})
		if getErr == nil {
			// Successfully retrieved session, check type
			spec, found, _ := unstructured.NestedMap(sessionObj.Object, "spec")
			if found {
				sessionType, _ := spec["type"].(string)

				if sessionType == "langflow" {
					// Extract messages from LangFlow session results
					log.Printf("GetSessionMessagesWS: LangFlow session detected, extracting messages from CR status")
					messages, err = extractMessagesFromLangFlowSession(sessionObj, sessionID)
				} else {
					// Default to S3 retrieval for Claude Code sessions
					log.Printf("GetSessionMessagesWS: Claude Code session, retrieving from S3")
					messages, err = retrieveMessagesFromS3(sessionID)
				}
			} else {
				// No spec found, default to S3
				messages, err = retrieveMessagesFromS3(sessionID)
			}
		} else {
			// Could not retrieve session (maybe doesn't exist or no permissions), try S3
			log.Printf("GetSessionMessagesWS: Could not retrieve session from K8s (%v), falling back to S3", getErr)
			messages, err = retrieveMessagesFromS3(sessionID)
		}
	} else {
		// DynamicClient not available, fall back to S3
		log.Printf("GetSessionMessagesWS: DynamicClient not available, using S3")
		messages, err = retrieveMessagesFromS3(sessionID)
	}

	if err != nil {
		log.Printf("getSessionMessagesWS: retrieve failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("failed to retrieve messages: %v", err),
		})
		return
	}

	// Optional consolidation of partial messages (for Claude Code sessions)
	includeParam := strings.ToLower(strings.TrimSpace(c.Query("include_partial_messages")))
	includePartials := includeParam == "1" || includeParam == "true" || includeParam == "yes"

	collapsed := make([]SessionMessage, 0, len(messages))
	activePartialIndex := -1
	for _, m := range messages {
		if m.Type == "message.partial" {
			if includePartials {
				if activePartialIndex >= 0 {
					collapsed[activePartialIndex] = m
				} else {
					collapsed = append(collapsed, m)
					activePartialIndex = len(collapsed) - 1
				}
			}
			// If not including partials, simply skip adding them
			continue
		}
		// On any non-partial, clear active partial placeholder
		activePartialIndex = -1
		collapsed = append(collapsed, m)
	}

	c.JSON(http.StatusOK, gin.H{
		"sessionId": sessionID,
		"messages":  collapsed,
	})
}

// PostSessionMessageWS handles POST /projects/:projectName/sessions/:sessionId/messages
// Accepts a generic JSON body. If a "type" string is provided, it will be used.
// Otherwise, defaults to "user_message" and wraps body under payload.
func PostSessionMessageWS(c *gin.Context) {
	sessionID := c.Param("sessionId")

	var body map[string]interface{}
	if err := c.BindJSON(&body); err != nil {
		log.Printf("postSessionMessageWS: bind failed: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON body"})
		return
	}

	msgType := "user_message"
	if v, ok := body["type"].(string); ok && v != "" {
		msgType = v
		// Remove type from payload to avoid duplication
		delete(body, "type")
	}

	message := &SessionMessage{
		SessionID: sessionID,
		Type:      msgType,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Payload:   body,
	}

	// Broadcast to session listeners (runner) and persist
	Hub.broadcast <- message

	c.JSON(http.StatusAccepted, gin.H{"status": "queued"})
}

// NOTE: GetSessionMessagesClaudeFormat removed - session continuation now uses
// SDK's built-in resume functionality with persisted ~/.claude state
// See: https://docs.claude.com/en/api/agent-sdk/sessions

// extractMessagesFromLangFlowSession extracts messages from a LangFlow session's CR status
// LangFlow results are stored in status.results.outputs as complex nested JSON
// This function transforms them into SessionMessage format for display in the UI
func extractMessagesFromLangFlowSession(sessionObj *unstructured.Unstructured, sessionID string) ([]SessionMessage, error) {
	messages := []SessionMessage{}

	// Extract status.results
	status, found, err := unstructured.NestedMap(sessionObj.Object, "status")
	if err != nil {
		return nil, fmt.Errorf("error accessing status: %w", err)
	}
	if !found {
		// No status yet, return empty messages
		return messages, nil
	}

	results, found := status["results"].(map[string]interface{})
	if !found {
		// No results yet, return empty messages
		return messages, nil
	}

	// Extract flowExecutionId if available
	flowExecutionID, _ := status["flowExecutionId"].(string)

	// Extract outputs array from results
	outputs, ok := results["outputs"].([]interface{})
	if !ok {
		log.Printf("LangFlow session %s: results.outputs is not an array", sessionID)
		return messages, nil
	}

	// Parse LangFlow's complex output structure
	// LangFlow returns: {outputs: [{inputs: {}, outputs: [{results: {...}}]}]}
	for idx, outputItem := range outputs {
		outputMap, ok := outputItem.(map[string]interface{})
		if !ok {
			continue
		}

		// Extract the nested outputs array
		nestedOutputs, ok := outputMap["outputs"].([]interface{})
		if !ok {
			continue
		}

		for nestedIdx, nestedOutput := range nestedOutputs {
			nestedMap, ok := nestedOutput.(map[string]interface{})
			if !ok {
				continue
			}

			// Extract results
			resultsData, ok := nestedMap["results"].(map[string]interface{})
			if !ok {
				continue
			}

			// Look for message data in results
			messageData, ok := resultsData["message"].(map[string]interface{})
			if !ok {
				continue
			}

			// Extract text content
			textContent, _ := messageData["text"].(string)
			sender, _ := messageData["sender"].(string)
			senderName, _ := messageData["sender_name"].(string)
			timestamp, _ := messageData["timestamp"].(string)

			// Default timestamp if not provided
			if timestamp == "" {
				timestamp = time.Now().UTC().Format(time.RFC3339)
			}

			// Create SessionMessage
			msg := SessionMessage{
				SessionID: sessionID,
				Type:      "langflow_output",
				Timestamp: timestamp,
				Payload: map[string]interface{}{
					"text":              textContent,
					"sender":            sender,
					"sender_name":       senderName,
					"flow_execution_id": flowExecutionID,
					"output_index":      idx,
					"nested_index":      nestedIdx,
					"component_id":      nestedMap["component_id"],
					"component_name":    nestedMap["component_display_name"],
				},
			}

			messages = append(messages, msg)
		}
	}

	// If no messages were extracted, create a summary message
	if len(messages) == 0 && len(outputs) > 0 {
		// Create a summary message showing the session completed
		completionTime, _ := status["completionTime"].(string)
		if completionTime == "" {
			completionTime = time.Now().UTC().Format(time.RFC3339)
		}

		summaryMsg := SessionMessage{
			SessionID: sessionID,
			Type:      "langflow_summary",
			Timestamp: completionTime,
			Payload: map[string]interface{}{
				"message":           "LangFlow execution completed",
				"flow_execution_id": flowExecutionID,
				"output_count":      len(outputs),
				"outputs":           outputs, // Include raw outputs for debugging
			},
		}
		messages = append(messages, summaryMsg)
	}

	log.Printf("Extracted %d messages from LangFlow session %s", len(messages), sessionID)
	return messages, nil
}
