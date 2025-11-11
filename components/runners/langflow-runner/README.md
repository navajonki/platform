# LangFlow Runner

Python-based runner for executing LangFlow workflows from AgenticSessions.

## Overview

This runner is spawned by the Agentic Operator when an AgenticSession with `type: langflow` is created. It:

1. Reads flow configuration from environment variables
2. Calls the LangFlow API to execute the flow
3. Updates the AgenticSession status with results

## Building

### For Minikube

```bash
# Build and load into Minikube
make load-minikube CONTAINER_ENGINE=docker

# Or with podman
make load-minikube CONTAINER_ENGINE=podman
```

### For Registry

```bash
# Build and push to quay.io
make push

# Custom registry
make push REGISTRY=docker.io/youruser
```

## Environment Variables

The runner expects the following environment variables (set by the operator):

| Variable | Required | Description |
|----------|----------|-------------|
| `FLOW_ID` | Yes | LangFlow flow ID to execute |
| `FLOW_INPUT` | No | JSON object with flow input parameters (default: `{}`) |
| `LANGFLOW_URL` | Yes | LangFlow API base URL |
| `LANGFLOW_API_KEY` | No | API key for LangFlow (optional for local dev) |
| `SESSION_NAME` | Yes | Name of the AgenticSession CR |
| `NAMESPACE` | Yes | Namespace of the AgenticSession CR |

## Flow Execution

The runner communicates with LangFlow via the REST API:

```
POST /api/v1/run/{flow_id}
Headers:
  Content-Type: application/json
  x-api-key: {api_key}
Body:
  {
    "inputs": {...},
    "output_type": "chat"
  }
```

Response is stored in the AgenticSession status:

```yaml
status:
  phase: Completed
  flowExecutionId: "exec-123"
  results:
    outputs: [...]
```

## Error Handling

The runner updates the session status based on execution outcome:

- **Success**: `phase: Completed` with results
- **LangFlow API Error**: `phase: Failed` with error message
- **Timeout**: `phase: Failed` after 1 hour
- **Invalid Input**: `phase: Failed` with validation error

## Local Testing

You can test the runner locally (without Kubernetes):

```bash
export FLOW_ID="your-flow-id"
export FLOW_INPUT='{"message": "test"}'
export LANGFLOW_URL="http://localhost:7860"
export LANGFLOW_API_KEY="your-api-key"
export SESSION_NAME="test-session"
export NAMESPACE="test"

# Run with mock k8s client (requires kubernetes config)
python3 run.py
```

## Dependencies

- `requests==2.31.0` - HTTP client for LangFlow API
- `kubernetes==28.1.0` - Kubernetes Python client for status updates

## Security

- Runs as non-root user (UID 1000)
- No privilege escalation
- API keys never logged (redacted in output)
- Minimal container image (python:3.11-slim)

## Permissions

The runner pod's ServiceAccount needs:

```yaml
rules:
- apiGroups: ["vteam.ambient-code"]
  resources: ["agenticsessions/status"]
  verbs: ["get", "patch"]
```

This is automatically configured by the operator when creating the Job.
