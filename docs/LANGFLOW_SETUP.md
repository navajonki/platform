# LangFlow Integration Setup Guide

This guide explains how to set up and use the LangFlow integration with Ambient Code Platform on Minikube.

## Overview

The LangFlow integration allows you to create and execute visual workflow automations as Ambient sessions. Users can:
1. Design flows visually in the LangFlow UI
2. Execute flows from the Ambient frontend
3. View results in the Ambient session interface

## Architecture

```
User → Ambient Frontend → Backend API → Operator → LangFlow Runner Pod → LangFlow Service
                                          ↓
                                    AgenticSession CR
```

**Components:**
- **LangFlow Service**: Visual workflow designer and execution engine (port 7860)
- **LangFlow Runner**: Python pod that executes flows via LangFlow API
- **Backend API**: Manages session lifecycle and LangFlow integration endpoints
- **Operator**: Watches AgenticSession CRs and creates LangFlow runner jobs

## Prerequisites

1. Minikube running with Docker driver
2. All Ambient Code Platform components deployed
3. PostgreSQL for LangFlow's database

## Deployment Steps

### 1. Deploy PostgreSQL for LangFlow

```bash
cd components/manifests
kubectl apply -f langflow/postgres-deployment.yaml
kubectl apply -f langflow/postgres-service.yaml
```

### 2. Deploy LangFlow

```bash
kubectl apply -f langflow/deployment.yaml
kubectl apply -f langflow/service.yaml
```

**Important Configuration:**
- `LANGFLOW_ENABLE_API_KEY=false` - API keys disabled for local development
- `LANGFLOW_AUTO_LOGIN=true` - Auto-login enabled
- `LANGFLOW_SKIP_AUTH_AUTO_LOGIN=true` - Skip auth for local dev

### 3. Build and Load LangFlow Runner Image

```bash
# Build the langflow-runner image
cd components/runners/langflow-runner
docker build -t localhost/vteam-langflow-runner:latest .

# Load into minikube
docker save localhost/vteam-langflow-runner:latest | docker exec -i minikube docker load
```

### 4. Configure Namespace

Each namespace where you want to use LangFlow sessions needs the managed label:

```bash
kubectl label namespace YOUR_PROJECT_NAME ambient-code.io/managed=true
```

### 5. Access LangFlow UI

**Option A: Ingress (Recommended)**

```bash
# Enable ingress addon
minikube addons enable ingress

# Apply ingress manifest
kubectl apply -f langflow/ingress.yaml

# Add to /etc/hosts
echo "$(minikube ip) langflow.local" | sudo tee -a /etc/hosts

# Start tunnel (in separate terminal)
minikube tunnel

# Access at: http://langflow.local
```

**Option B: NodePort (Quick)**

```bash
# Get URL
minikube service langflow -n ambient-code --url

# Access at returned URL (e.g., http://192.168.49.2:31234)
```

**Option C: Port-Forward (Temporary)**

```bash
kubectl port-forward svc/langflow 7860:7860 -n ambient-code

# Access at: http://localhost:7860
```

## Creating LangFlow Sessions

### 1. Design a Flow in LangFlow UI

1. Access LangFlow at http://langflow.local
2. Create a new flow or use existing flow
3. Add a "Chat Input" component with input name `input_value`
4. Add processing components (e.g., OpenAI)
5. Add a "Chat Output" component
6. Save the flow and note its Flow ID

### 2. Create Session from Ambient Frontend

1. Navigate to your project: `http://frontend.local/projects/YOUR_PROJECT/sessions/new`
2. Select "LangFlow" as session type
3. Choose your flow from the dropdown
4. Enter input value (will be passed to `input_value` field)
5. Click "Create Session"

### 3. Monitor Execution

The session will:
1. Create an AgenticSession CR
2. Operator creates a LangFlow runner Job
3. Runner executes the flow via LangFlow API
4. Results are stored in session status
5. Messages appear in the Ambient UI

## Frontend Configuration

The frontend needs proper environment variables:

```yaml
env:
- name: BACKEND_URL
  value: "http://backend-service:8080"  # No /api suffix!
- name: OC_TOKEN
  value: "YOUR_SERVICE_ACCOUNT_TOKEN"
- name: OC_USER
  value: "test-user"
- name: OC_EMAIL
  value: "test@example.com"
```

**Important:** `BACKEND_URL` must NOT include `/api` suffix. Next.js API routes add this automatically.

## Backend Configuration

The backend requires LangFlow client configuration:

```yaml
env:
- name: LANGFLOW_URL
  value: "http://langflow.ambient-code.svc.cluster.local:7860"
```

Backend endpoints:
- `GET /api/langflow/health` - Check LangFlow availability
- `GET /api/langflow/flows` - List all flows
- `GET /api/langflow/flows/:id` - Get specific flow

## Operator Configuration

The operator creates LangFlow runner jobs with these environment variables:

```yaml
- FLOW_ID: ID of the flow to execute
- FLOW_INPUT: JSON string of input values
- LANGFLOW_URL: http://langflow.ambient-code.svc.cluster.local:7860
- SESSION_NAME: Name of the AgenticSession
- NAMESPACE: Project namespace
- BACKEND_API_URL: Backend API for status updates
- BOT_TOKEN: ServiceAccount token for auth
```

**Note:** `LANGFLOW_API_KEY` is intentionally omitted since LangFlow has API keys disabled.

## Troubleshooting

### LangFlow service not available

**Symptoms:** Frontend shows "LangFlow service is not available"

**Checks:**
```bash
# 1. Verify LangFlow pod is running
kubectl get pods -n ambient-code -l app=langflow

# 2. Check LangFlow health
kubectl exec -it deployment/backend-api -n ambient-code -- \
  curl http://langflow.ambient-code.svc.cluster.local:7860/health

# 3. Check backend logs
kubectl logs -n ambient-code -l app=backend-api | grep langflow
```

### Session stuck in Pending

**Symptoms:** Session created but no job appears

**Checks:**
```bash
# 1. Verify namespace has managed label
kubectl get namespace YOUR_PROJECT -o jsonpath='{.metadata.labels}'

# 2. Add label if missing
kubectl label namespace YOUR_PROJECT ambient-code.io/managed=true

# 3. Check operator logs
kubectl logs -n ambient-code -l app=agentic-operator
```

### Session fails with API key error

**Symptoms:** Session status shows "Invalid or missing API key"

**Root Cause:** Operator sending API key when LangFlow has API keys disabled

**Fix:** Remove `LANGFLOW_API_KEY` env var from operator job spec (should be fixed in code)

### Runner image not found

**Symptoms:** Pod shows `ErrImageNeverPull` or `ImagePullBackOff`

**Fix:**
```bash
# Build and load the image
cd components/runners/langflow-runner
docker build -t localhost/vteam-langflow-runner:latest .
docker save localhost/vteam-langflow-runner:latest | docker exec -i minikube docker load
```

### PostgreSQL connection issues

**Symptoms:** LangFlow pod shows database connection errors

**Check deployment env vars:**
```yaml
env:
- name: POSTGRES_HOST
  value: "langflow-postgres"
- name: POSTGRES_PORT
  value: "5432"
- name: POSTGRES_USER
  valueFrom:
    secretKeyRef:
      name: langflow-postgres
      key: POSTGRES_USER
- name: POSTGRES_PASSWORD
  valueFrom:
    secretKeyRef:
      name: langflow-postgres
      key: POSTGRES_PASSWORD
```

**Note:** Do NOT use `LANGFLOW_DATABASE_URL` with shell-style variable expansion. Kubernetes doesn't expand `$(POSTGRES_USER)` syntax.

## Architecture Details

### Session Lifecycle

1. **Creation:**
   - User submits form in frontend
   - Frontend calls `POST /api/projects/:project/agentic-sessions`
   - Backend creates AgenticSession CR and ServiceAccount token secret

2. **Execution:**
   - Operator watches for AgenticSession with `type: langflow`
   - Operator waits for token secret to exist (race condition prevention)
   - Operator creates Job with langflow-runner pod

3. **Processing:**
   - Runner pod executes flow via LangFlow API
   - Runner updates session status via backend API
   - Job completes, pod terminates after 10 minutes (TTL)

4. **Completion:**
   - Session status updated to Completed/Failed
   - Results stored in `status.results`
   - Flow execution ID stored in `status.flowExecutionId`
   - Messages available via `/api/projects/:project/agentic-sessions/:name/messages`

### AgenticSession Spec

```yaml
apiVersion: vteam.ambient-code/v1alpha1
kind: AgenticSession
metadata:
  name: agentic-session-123456
  namespace: my-project
  labels:
    sessionType: langflow
spec:
  type: langflow
  flowId: "74c01208-551e-4fee-812e-d8fdcc73d3e1"
  flowInput:
    input_value: "User's input text"
  prompt: "LangFlow session: 74c01208-551e-4fee-812e-d8fdcc73d3e1"
  timeout: 300
```

### Status Fields

```yaml
status:
  phase: Completed  # Pending | Running | Completed | Failed
  jobName: agentic-session-123456-job
  startTime: "2025-11-18T18:00:00Z"
  completionTime: "2025-11-18T18:05:00Z"
  flowExecutionId: "langflow-session-abc123"
  results:
    outputs:
      - message: "Flow execution result"
        type: "text"
```

## Component Integration Points

### Frontend Components

- `src/app/projects/[name]/sessions/new/session-type-selector.tsx` - Radio group for session type
- `src/app/projects/[name]/sessions/new/langflow-selector.tsx` - Flow selection dropdown
- `src/app/projects/[name]/sessions/new/langflow-input-config.tsx` - Input field component
- `src/services/api/langflow.ts` - API client for LangFlow endpoints
- `src/services/queries/use-langflow.ts` - React Query hooks

### Backend Handlers

- `handlers/langflow.go` - LangFlow health and flow listing endpoints
- `langflow/client.go` - LangFlow API client implementation
- `types/session.go` - Extended session types with LangFlow fields

### Operator Handlers

- `internal/handlers/sessions.go:755` - `handleLangFlowSession()` function
- Creates Job with langflow-runner image
- Sets environment variables for runner
- Monitors job execution

### Runner Implementation

- `components/runners/langflow-runner/run.py` - Main execution script
- Calls LangFlow API to execute flow
- Updates session status via backend API
- Handles errors and timeouts

## Testing

### Manual Test Flow

```bash
# 1. Verify all components running
kubectl get pods -n ambient-code

# 2. Create a test project
# Via UI: http://frontend.local

# 3. Label the namespace
kubectl label namespace YOUR_PROJECT ambient-code.io/managed=true

# 4. Create a simple flow in LangFlow
# Access: http://langflow.local

# 5. Create session via Ambient UI
# Navigate to: http://frontend.local/projects/YOUR_PROJECT/sessions/new

# 6. Monitor execution
kubectl get agenticsession -n YOUR_PROJECT -w
kubectl logs -n YOUR_PROJECT -l session-type=langflow -f

# 7. Verify completion
kubectl get agenticsession SESSION_NAME -n YOUR_PROJECT -o yaml
```

## Production Considerations

For production deployments:

1. **Enable API Keys:**
   - Set `LANGFLOW_ENABLE_API_KEY=true`
   - Create API key in LangFlow UI
   - Store in Kubernetes secret
   - Update operator to pass API key env var

2. **Use External PostgreSQL:**
   - Deploy managed PostgreSQL instance
   - Update LangFlow deployment with connection details

3. **Configure Resource Limits:**
   - Set appropriate CPU/memory for LangFlow pod
   - Configure runner job resource requests/limits

4. **Enable Persistence:**
   - Create PersistentVolumeClaim for LangFlow data
   - Mount at `/data` in LangFlow pod

5. **Secure Ingress:**
   - Configure TLS certificates
   - Enable OAuth/authentication
   - Restrict access to authorized users

## Reference

- LangFlow Documentation: https://docs.langflow.org/
- Ambient Code Platform: See `README.md`
- Minikube Setup: See `docs/MINIKUBE_SETUP.md`
- Design Document: See `docs/design/langflow-integration.md`
