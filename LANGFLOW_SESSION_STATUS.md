# LangFlow Integration - Current Status

**Last Updated**: 2025-11-18 19:00 UTC
**Status**: ✅ **WORKING END-TO-END**

## Quick Access

### Access URLs (with minikube tunnel running)
- **Frontend**: http://frontend.local
- **LangFlow**: http://langflow.local
- **Backend API**: http://backend.local/api

### Minikube Cluster
- **Driver**: Docker Desktop
- **IP**: 192.168.49.2
- **Ingress**: Enabled with nginx controller
- **Access Method**: Ingress + minikube tunnel (stable)

## ✅ Working Components

### 1. LangFlow Service
- **Status**: Running (1/1)
- **Database**: PostgreSQL (working)
- **Configuration**:
  - `LANGFLOW_ENABLE_API_KEY=false` (local dev)
  - `LANGFLOW_AUTO_LOGIN=true`
  - Individual `POSTGRES_*` env vars (fixed from DATABASE_URL)
- **Access**: http://langflow.local

### 2. Backend API
- **Status**: Running (1/1)
- **Image**: vteam-backend:latest (local build)
- **LangFlow Endpoints**:
  - `GET /api/langflow/health` ✅
  - `GET /api/langflow/flows` ✅
  - `GET /api/langflow/flows/:id` ✅
- **Configuration**:
  - `BACKEND_URL=http://backend-service:8080` (no /api suffix!)
  - `LANGFLOW_URL=http://langflow.ambient-code.svc.cluster.local:7860`

### 3. Frontend
- **Status**: Running (1/1)
- **Image**: vteam-frontend:latest (local build)
- **Features**:
  - Session type selector (Claude Code / LangFlow)
  - LangFlow flow selector with live flow list
  - Input configuration for flow parameters
- **Configuration**:
  - `BACKEND_URL=http://backend-service:8080`
  - `OC_TOKEN`, `OC_USER`, `OC_EMAIL` for auth

### 4. Operator
- **Status**: Running (1/1)
- **Image**: vteam-operator:latest (local build with LangFlow support)
- **Features**:
  - Watches AgenticSessions with `type: langflow`
  - Creates Jobs with langflow-runner pods
  - Token secret race condition fix implemented
  - **Fixed**: Removed `LANGFLOW_API_KEY` env var (API keys disabled)

### 5. LangFlow Runner
- **Image**: localhost/vteam-langflow-runner:latest
- **Status**: Ready (image loaded in minikube)
- **Execution**: Creates Job pods that execute flows via LangFlow API

## 🔧 Critical Fixes Applied

### 1. Namespace Managed Label
**Problem**: Operator skipping namespaces without label
**Fix**: `kubectl label namespace PROJECT_NAME ambient-code.io/managed=true`
**Location**: operator/internal/handlers/sessions.go:57

### 2. Frontend BACKEND_URL
**Problem**: Double `/api` in URL path (frontend adding /api + BACKEND_URL with /api)
**Fix**: Changed `BACKEND_URL` from `http://backend-service:8080/api` to `http://backend-service:8080`
**Location**: frontend deployment env vars

### 3. LangFlow Database Connection
**Problem**: Shell-style `$(POSTGRES_USER)` expansion not working in K8s env vars
**Fix**: Changed from `LANGFLOW_DATABASE_URL` to individual `POSTGRES_HOST`, `POSTGRES_PORT`, etc.
**Location**: components/manifests/langflow/deployment.yaml:51-70

### 4. LangFlow API Key
**Problem**: Runner sending API key, LangFlow rejecting with 403 (API keys disabled)
**Fix**: Removed `LANGFLOW_API_KEY` env var from operator job spec
**Location**: operator/internal/handlers/sessions.go:844-859

### 5. Frontend Authentication
**Problem**: Frontend calls returning 401 Unauthorized
**Fix**: Added ServiceAccount token to frontend env (`OC_TOKEN`, `OC_USER`, `OC_EMAIL`)
**Location**: frontend deployment

### 6. Image Deployment
**Problem**: All images were old quay.io versions without LangFlow code
**Fix**: Rebuilt and loaded all images locally:
- Backend: `make build-backend CONTAINER_ENGINE=docker`
- Frontend: `make build-frontend CONTAINER_ENGINE=docker`
- Operator: `make build-operator CONTAINER_ENGINE=docker`
- LangFlow Runner: `docker build -t localhost/vteam-langflow-runner:latest .`

### 7. Messages Endpoint Support for Text Output
**Problem**: Messages endpoint only supported Chat Output components (`artifacts.message`), failed for Text Output components (`artifacts.text.raw`)
**Fix**: Updated backend handler to try both paths:
**Location**: `components/backend/handlers/sessions.go:2678-2689`
```go
// Try Chat Output format first (artifacts.message)
message, found, _ := unstructured.NestedString(artifacts, "message")

// If not found, try Text Output format (artifacts.text.raw)
if !found {
    message, found, _ = unstructured.NestedString(artifacts, "text", "raw")
}
```

### 8. Local Development Environment Variables
**Problem**: Backend deployment missing DISABLE_AUTH and ENVIRONMENT env vars for local dev
**Fix**: Added to backend-deployment.yaml:
**Location**: `components/manifests/base/backend-deployment.yaml:35-39`
```yaml
- name: DISABLE_AUTH
  value: "true"
- name: ENVIRONMENT
  value: "local"
```

## 📦 Container Images (in minikube)

```
vteam-backend:latest              - Backend with LangFlow endpoints
vteam-frontend:latest             - Frontend with LangFlow UI components
vteam-operator:latest             - Operator with LangFlow session handler
localhost/vteam-langflow-runner:latest - LangFlow runner
langflowai/langflow:latest       - LangFlow service
postgres:13                       - PostgreSQL for LangFlow
```

## 🎯 Testing End-to-End

### 1. Access LangFlow and Create Flow
```bash
# Access LangFlow UI
open http://langflow.local

# Create a flow with:
# - Chat Input (input_value)
# - Processing (e.g., OpenAI)
# - Chat Output

# Save flow and note Flow ID (e.g., 74c01208-551e-4fee-812e-d8fdcc73d3e1)
```

### 2. Create LangFlow Session

```bash
# 1. Access Ambient Frontend
open http://frontend.local

# 2. Navigate to project sessions page
# Example: http://frontend.local/projects/langflow-test/sessions/new

# 3. Select "LangFlow" session type

# 4. Choose flow from dropdown (flows are fetched from LangFlow API)

# 5. Enter input value (e.g., "90+90")

# 6. Click "Create Session"
```

### 3. Monitor Execution

```bash
# Watch session status
kubectl get agenticsession -n langflow-test -w

# Check operator logs
kubectl logs -n ambient-code -l app=agentic-operator -f

# Check runner job
kubectl get jobs -n langflow-test
kubectl get pods -n langflow-test

# View runner logs
kubectl logs -n langflow-test -l session-type=langflow -f
```

### 4. Verify Completion

```bash
# Check session status
kubectl get agenticsession SESSION_NAME -n langflow-test -o yaml

# Should show:
# status:
#   phase: Completed
#   flowExecutionId: "langflow-session-xyz"
#   results:
#     outputs: [...]
```

## 📝 Architecture Flow

### Session Creation
1. User submits form in frontend (http://frontend.local)
2. Frontend calls `POST /api/projects/{project}/agentic-sessions`
3. Backend creates AgenticSession CR and ServiceAccount token secret
4. Backend returns session info to frontend

### Session Execution
5. Operator watches AgenticSession with `type: langflow`
6. Operator waits for token secret to exist (race condition prevention)
7. Operator creates Job with langflow-runner pod
8. Runner pod executes flow via LangFlow API (`POST /api/v1/run/{flowId}`)
9. Runner updates session status via backend API (`PUT /api/projects/{project}/agentic-sessions/{name}/status`)
10. Job completes, pod terminates (TTL: 10 minutes)

### Result Display
11. Frontend polls session status
12. Session shows phase: Completed
13. Frontend fetches messages (`GET /api/projects/{project}/agentic-sessions/{name}/messages`)
14. Messages display in UI with LangFlow output

## 🔑 Key Configuration

### Namespace Requirements
- Label: `ambient-code.io/managed=true` (required for operator to process)

### Frontend Environment
```yaml
BACKEND_URL: "http://backend-service:8080"  # NO /api suffix
OC_TOKEN: "SERVICE_ACCOUNT_TOKEN"
OC_USER: "test-user"
OC_EMAIL: "test@example.com"
```

### Backend Environment
```yaml
LANGFLOW_URL: "http://langflow.ambient-code.svc.cluster.local:7860"
```

### LangFlow Environment
```yaml
LANGFLOW_ENABLE_API_KEY: "false"  # Disabled for local dev
LANGFLOW_AUTO_LOGIN: "true"
POSTGRES_HOST: "langflow-postgres"
POSTGRES_PORT: "5432"
# Individual POSTGRES_* vars, NOT LANGFLOW_DATABASE_URL
```

### Runner Job Environment (set by operator)
```yaml
FLOW_ID: "flow-id-from-session-spec"
FLOW_INPUT: '{"input_value":"user input"}'
LANGFLOW_URL: "http://langflow.ambient-code.svc.cluster.local:7860"
SESSION_NAME: "agentic-session-123456"
NAMESPACE: "project-namespace"
BACKEND_API_URL: "http://backend-service.ambient-code.svc.cluster.local:8080/api"
BOT_TOKEN: "from-secret-ambient-runner-token-{sessionName}"
```

## 🚀 Setup from Scratch

See detailed instructions in `docs/MINIKUBE_SETUP.md` and `docs/LANGFLOW_SETUP.md`.

**Quick start:**
```bash
# 1. Start minikube
minikube start --driver=docker
minikube addons enable ingress

# 2. Build images
make build-all CONTAINER_ENGINE=docker
cd components/runners/langflow-runner && docker build -t localhost/vteam-langflow-runner:latest .

# 3. Load images
docker save vteam-backend:latest | docker exec -i minikube docker load
docker save vteam-frontend:latest | docker exec -i minikube docker load
docker save vteam-operator:latest | docker exec -i minikube docker load
docker save localhost/vteam-langflow-runner:latest | docker exec -i minikube docker load

# 4. Deploy
cd components/manifests
kubectl apply -f base/namespace.yaml base/crds/ base/rbac/
kubectl apply -f langflow/
kubectl apply -f base/

# 5. Configure access
echo "127.0.0.1 langflow.local backend.local frontend.local" | sudo tee -a /etc/hosts
minikube tunnel  # Keep running

# 6. Create project and label namespace
kubectl label namespace YOUR_PROJECT ambient-code.io/managed=true
```

## 📚 Documentation

- **Setup Guide**: `docs/MINIKUBE_SETUP.md`
- **LangFlow Integration**: `docs/LANGFLOW_SETUP.md`
- **Design Doc**: `docs/design/langflow-integration.md`
- **CLAUDE.md**: Project instructions and patterns

## 🎉 Summary

**Status**: ✅ LangFlow integration is fully operational!

**What Works:**
- ✅ LangFlow UI accessible at http://langflow.local
- ✅ Flow creation and editing in LangFlow
- ✅ Flow listing in Ambient frontend dropdown
- ✅ Session creation from Ambient UI
- ✅ Job creation by operator
- ✅ Flow execution via LangFlow API
- ✅ Status updates to backend
- ✅ Results displayed in Ambient UI
- ✅ Messages endpoint returns LangFlow output (both Chat Output and Text Output components)

**Ready for:**
- Creating and executing LangFlow workflows
- Testing complex multi-step flows
- Integrating with OpenAI and other LangFlow components
- Production deployment (with API key configuration)
