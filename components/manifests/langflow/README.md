# LangFlow Deployment for Minikube

## Overview

This directory contains Kubernetes manifests for deploying LangFlow alongside the Ambient Code Platform on Minikube.

## Installation

### 1. Deploy LangFlow

```bash
# Apply all manifests
kubectl apply -f components/manifests/langflow/

# Wait for deployment to be ready
kubectl wait --for=condition=available deployment/langflow -n ambient-code --timeout=300s

# Check pod status
kubectl get pods -n ambient-code -l app=langflow
```

### 2. Access LangFlow UI

Add to `/etc/hosts`:
```
127.0.0.1 langflow.local
```

Then access at:
- **Minikube with Docker**: http://langflow.local
- **Minikube with Podman**: http://langflow.local:8080

Or use port-forward:
```bash
kubectl port-forward svc/langflow 7860:7860 -n ambient-code
# Access at http://localhost:7860
```

### 3. Configure API Key

1. Access LangFlow UI
2. Create an account or login
3. Go to Settings → API Keys
4. Generate a new API key
5. Update the secret:

```bash
kubectl create secret generic langflow-secret \
  --from-literal=api-key=YOUR_API_KEY \
  -n ambient-code \
  --dry-run=client -o yaml | kubectl apply -f -
```

### 4. Configure Backend

The backend needs to know where LangFlow is running. Update backend deployment:

```bash
kubectl set env deployment/backend-api \
  LANGFLOW_URL=http://langflow.ambient-code.svc.cluster.local:7860 \
  -n ambient-code

# Also add API key from secret
kubectl patch deployment backend-api -n ambient-code --type=json -p='[
  {
    "op": "add",
    "path": "/spec/template/spec/containers/0/env/-",
    "value": {
      "name": "LANGFLOW_API_KEY",
      "valueFrom": {
        "secretKeyRef": {
          "name": "langflow-secret",
          "key": "api-key"
        }
      }
    }
  }
]'
```

## Verification

### Check LangFlow Health

```bash
# From within cluster
kubectl run -it --rm debug --image=curlimages/curl --restart=Never -- \
  curl http://langflow.ambient-code.svc.cluster.local:7860/health

# Expected output: {"status":"healthy"}
```

### List Flows (with API key)

```bash
kubectl run -it --rm debug --image=curlimages/curl --restart=Never -- \
  curl -H "x-api-key: YOUR_API_KEY" \
  http://langflow.ambient-code.svc.cluster.local:7860/api/v1/flows
```

## Troubleshooting

### Pod not starting

```bash
# Check pod status
kubectl describe pod -n ambient-code -l app=langflow

# Check logs
kubectl logs -n ambient-code -l app=langflow --tail=100
```

### Storage issues

If PVC is pending:
```bash
kubectl get pvc -n ambient-code langflow-data

# Check storage class exists
kubectl get storageclass
```

Minikube should have a `standard` storage class by default.

### Health check failing

LangFlow can take 30-60 seconds to start up. Check logs:
```bash
kubectl logs -n ambient-code deployment/langflow -f
```

## Cleanup

```bash
# Remove LangFlow deployment
kubectl delete -f components/manifests/langflow/

# Delete PVC and data (WARNING: destroys all flows)
kubectl delete pvc langflow-data -n ambient-code
```

## Architecture Notes

- **Database**: SQLite stored in PVC (single-instance only)
- **API Access**: ClusterIP service for internal communication
- **UI Access**: Ingress for external access
- **Authentication**: API key stored in Kubernetes Secret
- **Storage**: 5Gi PVC for flow storage and database

For production deployments, consider:
- PostgreSQL instead of SQLite
- Multiple replicas with shared storage
- Proper backup strategy
- Resource limits tuning
