# Getting the Ambient Code Platform Running on Minikube

## Summary of Setup Steps

Here's a complete guide to get the platform running on Minikube from the `feat/migrate-crc-to-minikube` branch.

### Prerequisites
```bash
# Required tools
brew install minikube kubectl podman docker
```

### 1. Start Minikube
```bash
minikube start --driver=docker
# or
minikube start --driver=podman
```

### 2. Build Container Images

Build all component images from the project root:

```bash
# Navigate to components
cd /path/to/platform

# Build backend
cd components/backend
podman build -q -t vteam-backend:latest .

# Build frontend
cd ../frontend
podman build -q -t vteam-frontend:latest .

# Build operator
cd ../operator
podman build -q -t vteam-operator:latest .

# Build Claude Code runner (from runners directory!)
cd ../runners
podman build -q -t vteam-claude-runner:latest -f claude-code-runner/Dockerfile .
```

### 3. Load Images into Minikube

```bash
# Save and load each image
podman save localhost/vteam-backend:latest -o /tmp/backend.tar
minikube image load /tmp/backend.tar

podman save localhost/vteam-frontend:latest -o /tmp/frontend.tar
minikube image load /tmp/frontend.tar

podman save localhost/vteam-operator:latest -o /tmp/operator.tar
minikube image load /tmp/operator.tar

podman save localhost/vteam-claude-runner:latest -o /tmp/claude-runner.tar
minikube image load /tmp/claude-runner.tar

# Retag images without localhost/ prefix inside minikube
minikube ssh "docker tag localhost/vteam-backend:latest vteam-backend:latest"
minikube ssh "docker tag localhost/vteam-frontend:latest vteam-frontend:latest"
minikube ssh "docker tag localhost/vteam-operator:latest vteam-operator:latest"
minikube ssh "docker tag localhost/vteam-claude-runner:latest vteam-claude-runner:latest"
```

### 4. Deploy Platform Components

```bash
cd components/manifests

# Apply base manifests
kubectl apply -f base/namespace.yaml
kubectl apply -f base/crds/
kubectl apply -f base/rbac/
kubectl apply -f minikube/local-dev-rbac.yaml

# Deploy services
kubectl apply -f minikube/backend-deployment.yaml
kubectl apply -f minikube/frontend-deployment.yaml
kubectl apply -f minikube/operator-deployment.yaml
kubectl apply -f minikube/backend-service.yaml
kubectl apply -f minikube/frontend-service.yaml

# Wait for pods to be ready
kubectl wait --for=condition=ready pod -l app=backend-api -n ambient-code --timeout=120s
kubectl wait --for=condition=ready pod -l app=frontend -n ambient-code --timeout=120s
kubectl wait --for=condition=ready pod -l app=agentic-operator -n ambient-code --timeout=120s
```

### 5. Access the Frontend

```bash
# Get the frontend URL
minikube service frontend-service -n ambient-code --url
# Example output: http://192.168.64.4:30030

# Open in browser
open $(minikube service frontend-service -n ambient-code --url)
```

### 6. Create a Project

The UI should now load. Create a new project (e.g., "test-web").

### 7. Configure GitHub Token

For each project namespace, create a secret with your GitHub Personal Access Token:

```bash
# Create GitHub PAT at: https://github.com/settings/tokens
# Required scopes: repo (for private repos) or public_repo (for public repos)

kubectl create secret generic ambient-runner-secrets \
  -n YOUR_PROJECT_NAME \
  --from-literal=GIT_TOKEN=YOUR_GITHUB_TOKEN \
  --from-literal=ANTHROPIC_API_KEY=YOUR_ANTHROPIC_API_KEY
```

### 8. Configure ProjectSettings

Each project needs its ProjectSettings CR configured to use the runner secrets:

```bash
kubectl patch projectsettings projectsettings -n YOUR_PROJECT_NAME \
  --type=merge \
  -p '{"spec":{"runnerSecretsName":"ambient-runner-secrets"}}'
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
