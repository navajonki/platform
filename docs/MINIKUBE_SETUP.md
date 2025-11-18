# Getting the Ambient Code Platform Running on Minikube

## Summary of Setup Steps

Here's a complete guide to get the platform running on Minikube with LangFlow integration.

### Prerequisites
```bash
# Required tools
brew install minikube kubectl docker
# Or use podman instead of docker
# brew install minikube kubectl podman
```

### 1. Start Minikube
```bash
# Start with Docker driver (recommended)
minikube start --driver=docker

# Or use podman driver
# minikube start --driver=podman

# Enable ingress addon for stable access
minikube addons enable ingress
```

### 2. Build Container Images

Build all component images using the Makefile:

```bash
# Navigate to project root
cd /path/to/platform

# Build all images (uses docker by default)
make build-all CONTAINER_ENGINE=docker

# Or build individually
make build-backend CONTAINER_ENGINE=docker
make build-frontend CONTAINER_ENGINE=docker
make build-operator CONTAINER_ENGINE=docker
make build-runner CONTAINER_ENGINE=docker  # Claude Code runner

# Build LangFlow runner
cd components/runners/langflow-runner
docker build -t localhost/vteam-langflow-runner:latest .
cd ../../..
```

### 3. Load Images into Minikube

```bash
# Load images directly into minikube's Docker daemon
docker save vteam-backend:latest | docker exec -i minikube docker load
docker save vteam-frontend:latest | docker exec -i minikube docker load
docker save vteam-operator:latest | docker exec -i minikube docker load
docker save vteam-runner:latest | docker exec -i minikube docker load
docker save localhost/vteam-langflow-runner:latest | docker exec -i minikube docker load

# Verify images loaded
minikube ssh docker images | grep vteam
```

### 4. Deploy Platform Components

```bash
cd components/manifests

# Apply base manifests
kubectl apply -f base/namespace.yaml
kubectl apply -f base/crds/
kubectl apply -f base/rbac/

# Deploy PostgreSQL for LangFlow
kubectl apply -f langflow/postgres-secret.yaml
kubectl apply -f langflow/postgres-deployment.yaml
kubectl apply -f langflow/postgres-service.yaml

# Deploy LangFlow
kubectl apply -f langflow/deployment.yaml
kubectl apply -f langflow/service.yaml
kubectl apply -f langflow/ingress.yaml

# Deploy backend, frontend, operator
kubectl apply -f base/backend-deployment.yaml
kubectl apply -f base/backend-service.yaml
kubectl apply -f base/frontend-deployment.yaml
kubectl apply -f base/frontend-service.yaml
kubectl apply -f base/frontend-ingress.yaml
kubectl apply -f base/operator-deployment.yaml

# Wait for pods to be ready
kubectl wait --for=condition=ready pod -l app=langflow-postgres -n ambient-code --timeout=180s
kubectl wait --for=condition=ready pod -l app=langflow -n ambient-code --timeout=180s
kubectl wait --for=condition=ready pod -l app=backend-api -n ambient-code --timeout=120s
kubectl wait --for=condition=ready pod -l app=frontend -n ambient-code --timeout=120s
kubectl wait --for=condition=ready pod -l app=agentic-operator -n ambient-code --timeout=120s
```

### 5. Configure /etc/hosts and Start Tunnel

```bash
# Add entries to /etc/hosts
echo "127.0.0.1 langflow.local backend.local frontend.local" | sudo tee -a /etc/hosts

# Start minikube tunnel (in separate terminal - keep it running)
minikube tunnel
# Enter your password when prompted

# Verify access
curl http://langflow.local/health
curl http://backend.local/api/cluster-info
curl http://frontend.local
```

### 6. Access the Services

Open in your browser:
- **Frontend UI**: http://frontend.local
- **LangFlow UI**: http://langflow.local
- **Backend API**: http://backend.local/api/cluster-info

### 7. Create a Project

1. Access http://frontend.local
2. Create a new project (e.g., "langflow-test")
3. The system will create a namespace with the same name

**Important:** Add the managed label to use LangFlow:
```bash
kubectl label namespace langflow-test ambient-code.io/managed=true
```

### 8. Configure Secrets

For each project namespace, create secrets:

```bash
# Create GitHub PAT at: https://github.com/settings/tokens
# Required scopes: repo (for private repos) or public_repo (for public repos)

kubectl create secret generic ambient-runner-secrets \
  -n YOUR_PROJECT_NAME \
  --from-literal=GIT_TOKEN=YOUR_GITHUB_TOKEN \
  --from-literal=ANTHROPIC_API_KEY=YOUR_ANTHROPIC_API_KEY

# For LangFlow sessions, copy the langflow-secret (if API keys are enabled)
# Note: Currently API keys are disabled for local dev, so this is optional
kubectl get secret langflow-secret -n ambient-code -o yaml | \
  sed 's/namespace: ambient-code/namespace: YOUR_PROJECT_NAME/' | \
  kubectl apply -f -
```

### 9. Configure ProjectSettings

Each project needs its ProjectSettings CR configured to use the runner secrets:

```bash
kubectl patch projectsettings projectsettings -n YOUR_PROJECT_NAME \
  --type=merge \
  -p '{"spec":{"runnerSecretsName":"ambient-runner-secrets"}}'
```

### 10. Create LangFlow Sessions

1. Access LangFlow UI at http://langflow.local
2. Create a new flow with:
   - Chat Input component (input name: `input_value`)
   - Processing components (e.g., OpenAI, prompts)
   - Chat Output component
3. Save the flow and note its ID

4. In Ambient UI (http://frontend.local):
   - Navigate to your project → Sessions → New Session
   - Select "LangFlow" as session type
   - Choose your flow from dropdown
   - Enter input text
   - Click "Create Session"

5. Monitor execution:
   ```bash
   kubectl get agenticsession -n YOUR_PROJECT_NAME -w
   kubectl logs -n YOUR_PROJECT_NAME -l session-type=langflow -f
   ```

## Key Fixes Applied to This Branch

### 1. Local Dev Authentication Bypass

**File:** `components/backend/server/server.go`

Added automatic user identity injection when `DISABLE_AUTH=true`:

```go
if disableAuth == "true" {
    c.Set("userID", "local-dev-user")
    c.Set("userName", "local-dev-user")
    c.Set("userEmail", "local-dev@example.com")
    c.Next()
    return
}
```

### 2. Token Injection for Project Context

**File:** `components/backend/handlers/middleware.go`

Added mock token injection in `ValidateProjectContext` middleware for local dev:

```go
// In local dev mode, inject mock token if no token present
if isLocalDevEnvironment() && c.GetHeader("Authorization") == "" && c.GetHeader("X-Forwarded-Access-Token") == "" {
    c.Request.Header.Set("Authorization", "Bearer mock-token-for-local-dev")
    log.Printf("Local dev mode: injecting mock token for %s", c.FullPath())
}
```

### 3. RBAC Permissions

**File:** `components/manifests/base/rbac/backend-clusterrole.yaml`

Added ProjectSettings permissions:

```yaml
# ProjectSettings custom resources (for project access validation)
- apiGroups: ["vteam.ambient-code"]
  resources: ["projectsettings"]
  verbs: ["get", "list", "watch"]
```

## Environment Variables

The backend deployment uses these critical environment variables:

```yaml
env:
- name: DISABLE_AUTH
  value: "true"
- name: ENVIRONMENT
  value: "local"
- name: NAMESPACE
  value: "ambient-code"
```

## Troubleshooting

### Images not pulling
- Ensure `imagePullPolicy: Never` is set in deployment manifests
- Retag images without `localhost/` prefix inside minikube
- Verify images exist: `minikube image ls`

### Sessions failing with "User token required"
- Check ProjectSettings has `runnerSecretsName` configured
- Verify secret exists in project namespace

### Sessions failing with "ANTHROPIC_API_KEY required"
- Add ANTHROPIC_API_KEY to `ambient-runner-secrets` in each project namespace
- Ensure ProjectSettings points to the correct secret name

### RFE seeding fails
- Verify GIT_TOKEN in secret has correct permissions
- Use a real GitHub repository URL (not test/placeholder URLs)

## Architecture Notes

- **Projects = Kubernetes Namespaces**: Each project creates a new namespace with `ambient-code.io/managed=true` label
- **Sessions = Custom Resources**: AgenticSessions are CRs that spawn Jobs
- **Secrets are namespace-scoped**: Each project needs its own `ambient-runner-secrets`
- **ProjectSettings CR**: Required for each project, stores configuration like `runnerSecretsName`

## Quick Commands Reference

```bash
# View all projects (namespaces)
kubectl get namespaces -l ambient-code.io/managed=true

# View sessions in a project
kubectl get agenticsessions -n PROJECT_NAME

# View session status
kubectl get agenticsession SESSION_NAME -n PROJECT_NAME -o yaml

# View runner pod logs
kubectl logs -n PROJECT_NAME -l agentic-session=SESSION_NAME -c ambient-content

# Restart backend
kubectl rollout restart deployment/backend-api -n ambient-code

# Restart operator
kubectl rollout restart deployment/agentic-operator -n ambient-code

# Clean up a project
kubectl delete namespace PROJECT_NAME
```
