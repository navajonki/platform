# LangFlow Integration Design

## Overview

This document outlines the design and implementation plan for integrating LangFlow as an alternate flow orchestration mechanism within the Ambient Code Platform. This integration will enable users to create visual AI agent workflows using LangFlow's drag-and-drop interface while maintaining unified execution tracking through the platform's AgenticSession architecture.

## Background

### Current State

The Ambient Code Platform currently uses markdown-based agent definitions stored in the `agents/` directory. Each agent is defined with:
- YAML frontmatter (name, description, role, tools)
- Markdown content describing persona and behavior
- Execution via Claude Code SDK in runner pods

### Goals

1. Enable visual workflow creation using LangFlow
2. Support both markdown agents (existing) and LangFlow flows
3. Maintain unified monitoring and execution tracking
4. Leverage LangFlow's full feature set without reimplementation
5. Keep architecture simple and maintainable

## Architecture

### High-Level Design

```
┌─────────────────────────────────────────────────────────────┐
│                    Ambient Code Platform                     │
├─────────────────────────────────────────────────────────────┤
│                                                               │
│  ┌──────────┐      ┌──────────┐      ┌───────────┐         │
│  │ Frontend │─────▶│ Backend  │─────▶│ Operator  │         │
│  │          │      │   API    │      │           │         │
│  └──────────┘      └────┬─────┘      └─────┬─────┘         │
│                         │                    │               │
│                         ▼                    ▼               │
│                  ┌─────────────┐      ┌──────────┐          │
│                  │  LangFlow   │      │ Runner   │          │
│                  │   Client    │      │  Jobs    │          │
│                  └──────┬──────┘      └─────┬────┘          │
│                         │                    │               │
└─────────────────────────┼────────────────────┼───────────────┘
                          │                    │
                          ▼                    ▼
                   ┌─────────────────────────┐
                   │   LangFlow Service      │
                   │   (Separate Deployment) │
                   ├─────────────────────────┤
                   │ - Flow Builder UI       │
                   │ - Flow Storage          │
                   │ - Execution Runtime     │
                   │ - REST API (:7860/api)  │
                   └─────────────────────────┘
```

### Integration Approach

**Selected Option: LangFlow as Kubernetes Service with API Integration**

LangFlow will run as a separate service in the Kubernetes cluster, exposing its REST API for:
- Flow discovery (listing available flows)
- Flow metadata retrieval
- Flow execution
- Results collection

### Why This Approach?

**Considered Alternatives:**

1. **Export flows as JSON → Convert to Markdown Agents**
   - ❌ Complex conversion logic
   - ❌ Loses LangFlow's native runtime capabilities
   - ❌ Fragile maintenance burden

2. **Export flows as Python code**
   - ❌ Feature doesn't exist yet (GitHub issue #8272)
   - ❌ Would break with LangFlow updates
   - ❌ Still requires LangFlow dependencies

3. **LangFlow as primary orchestrator**
   - ❌ Inverts control flow
   - ❌ Complex bidirectional communication
   - ❌ Security implications

**Selected Approach Benefits:**

✅ Flows run in native LangFlow environment (full feature support)
✅ Clean separation of concerns
✅ Leverages existing LangFlow API
✅ No translation layer to maintain
✅ Kubernetes-native deployment pattern
✅ Both platforms can evolve independently

## Technical Design

### 1. AgenticSession CRD Extension

Extend the AgenticSession Custom Resource to support different execution types:

```yaml
apiVersion: vteam.ambient-code/v1alpha1
kind: AgenticSession
metadata:
  name: my-langflow-session
  namespace: my-project
spec:
  # NEW: Execution type
  type: "langflow"  # Options: "claude-code" (default), "langflow"

  # NEW: LangFlow-specific fields
  flowId: "abc-123-def-456"
  flowInput:
    message: "Analyze this repository"
    repository_url: "https://github.com/example/repo"

  # Existing fields (still supported)
  prompt: "..."
  repos: [...]
  interactive: false
  timeout: 3600
  model: "claude-sonnet-4"

status:
  phase: "Running"
  flowExecutionId: "exec-789"  # NEW: LangFlow execution tracking
  results:
    outputs: {}  # LangFlow flow outputs
  # ... existing status fields
```

**Backward Compatibility:**
- `type` defaults to `"claude-code"` if not specified
- Existing sessions continue to work without modification

### 2. Backend API Changes

#### New LangFlow Client

**File:** `components/backend/langflow/client.go`

```go
package langflow

import (
    "bytes"
    "encoding/json"
    "fmt"
    "net/http"
)

type Client struct {
    BaseURL string
    APIKey  string
    client  *http.Client
}

type Flow struct {
    ID          string                 `json:"id"`
    Name        string                 `json:"name"`
    Description string                 `json:"description"`
    Data        map[string]interface{} `json:"data"`
    UpdatedAt   string                 `json:"updated_at"`
}

type FlowExecutionRequest struct {
    InputValue string                 `json:"input_value,omitempty"`
    Tweaks     map[string]interface{} `json:"tweaks,omitempty"`
    Inputs     map[string]interface{} `json:"inputs,omitempty"`
}

type FlowExecutionResponse struct {
    SessionID string                 `json:"session_id"`
    Outputs   []FlowOutput           `json:"outputs"`
}

type FlowOutput struct {
    Type    string                 `json:"type"`
    Message string                 `json:"message,omitempty"`
    Data    map[string]interface{} `json:"data"`
}

func NewClient(baseURL, apiKey string) *Client {
    return &Client{
        BaseURL: baseURL,
        APIKey:  apiKey,
        client:  &http.Client{},
    }
}

func (c *Client) ListFlows() ([]Flow, error) {
    // GET /api/v1/flows
}

func (c *Client) GetFlow(id string) (*Flow, error) {
    // GET /api/v1/flows/{id}
}

func (c *Client) RunFlow(id string, req FlowExecutionRequest) (*FlowExecutionResponse, error) {
    // POST /api/v1/run/{id}
}

func (c *Client) DownloadFlow(id string) ([]byte, error) {
    // GET /api/v1/flows/download/{id}
}
```

#### New API Endpoints

**File:** `components/backend/handlers/langflow.go`

```go
package handlers

// GET /api/langflow/flows
func ListLangFlowFlows(c *gin.Context) {
    flows, err := langflowClient.ListFlows()
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch flows"})
        return
    }
    c.JSON(http.StatusOK, gin.H{"flows": flows})
}

// GET /api/langflow/flows/:id
func GetLangFlowFlow(c *gin.Context) {
    flowID := c.Param("id")
    flow, err := langflowClient.GetFlow(flowID)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "Flow not found"})
        return
    }
    c.JSON(http.StatusOK, flow)
}

// GET /api/langflow/health
func LangFlowHealth(c *gin.Context) {
    // Check if LangFlow service is reachable
}
```

**Route Registration:** `components/backend/routes.go`

```go
// LangFlow integration endpoints
r.GET("/api/langflow/health", LangFlowHealth)
r.GET("/api/langflow/flows", ListLangFlowFlows)
r.GET("/api/langflow/flows/:id", GetLangFlowFlow)
```

#### Session Creation Updates

**File:** `components/backend/handlers/sessions.go`

Update `CreateAgenticSession` to validate LangFlow-specific fields:

```go
func CreateAgenticSession(c *gin.Context) {
    // ... existing code ...

    // NEW: Validate type-specific fields
    sessionType := spec["type"].(string)
    if sessionType == "langflow" {
        flowID, ok := spec["flowId"].(string)
        if !ok || flowID == "" {
            c.JSON(http.StatusBadRequest, gin.H{"error": "flowId required for langflow sessions"})
            return
        }

        // Validate flow exists
        _, err := langflowClient.GetFlow(flowID)
        if err != nil {
            c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid flowId"})
            return
        }
    }

    // ... rest of existing code ...
}
```

### 3. Operator Changes

#### LangFlow Session Handler

**File:** `components/operator/internal/handlers/langflow_sessions.go`

```go
package handlers

import (
    "k8s.io/api/batch/v1"
    corev1 "k8s.io/api/core/v1"
)

func HandleLangFlowSession(obj *unstructured.Unstructured) error {
    name := obj.GetName()
    namespace := obj.GetNamespace()

    // Extract spec fields
    spec, _, _ := unstructured.NestedMap(obj.Object, "spec")
    flowID := spec["flowId"].(string)
    flowInput := spec["flowInput"].(map[string]interface{})

    // Create Job
    job := createLangFlowJob(namespace, name, flowID, flowInput, obj)

    _, err := K8sClient.BatchV1().Jobs(namespace).Create(ctx, job, v1.CreateOptions{})
    if err != nil {
        log.Printf("Failed to create LangFlow job: %v", err)
        return err
    }

    // Update status
    updateAgenticSessionStatus(namespace, name, map[string]interface{}{
        "phase": "Running",
    })

    // Start monitoring goroutine
    go monitorLangFlowJob(job.Name, name, namespace)

    return nil
}

func createLangFlowJob(namespace, sessionName, flowID string, flowInput map[string]interface{}, owner *unstructured.Unstructured) *batchv1.Job {
    // Convert flowInput to JSON string
    inputJSON, _ := json.Marshal(flowInput)

    return &batchv1.Job{
        ObjectMeta: metav1.ObjectMeta{
            Name:      fmt.Sprintf("langflow-%s", sessionName),
            Namespace: namespace,
            OwnerReferences: []metav1.OwnerReference{
                {
                    APIVersion: owner.GetAPIVersion(),
                    Kind:       owner.GetKind(),
                    Name:       owner.GetName(),
                    UID:        owner.GetUID(),
                    Controller: boolPtr(true),
                },
            },
        },
        Spec: batchv1.JobSpec{
            Template: corev1.PodTemplateSpec{
                Spec: corev1.PodSpec{
                    RestartPolicy: corev1.RestartPolicyNever,
                    Containers: []corev1.Container{
                        {
                            Name:  "langflow-runner",
                            Image: "quay.io/ambient_code/langflow_runner:latest",
                            Env: []corev1.EnvVar{
                                {Name: "FLOW_ID", Value: flowID},
                                {Name: "FLOW_INPUT", Value: string(inputJSON)},
                                {Name: "LANGFLOW_URL", Value: "http://langflow.ambient-code.svc.cluster.local:7860"},
                                {Name: "LANGFLOW_API_KEY", ValueFrom: &corev1.EnvVarSource{
                                    SecretKeyRef: &corev1.SecretKeySelector{
                                        LocalObjectReference: corev1.LocalObjectReference{Name: "langflow-secret"},
                                        Key: "api-key",
                                    },
                                }},
                                {Name: "SESSION_NAME", Value: sessionName},
                                {Name: "NAMESPACE", Value: namespace},
                            },
                            SecurityContext: &corev1.SecurityContext{
                                AllowPrivilegeEscalation: boolPtr(false),
                                Capabilities: &corev1.Capabilities{
                                    Drop: []corev1.Capability{"ALL"},
                                },
                            },
                        },
                    },
                },
            },
        },
    }
}
```

#### Session Type Routing

**File:** `components/operator/internal/handlers/sessions.go`

Update the main session handler to route by type:

```go
func handleAgenticSessionEvent(obj *unstructured.Unstructured) error {
    spec, _, _ := unstructured.NestedMap(obj.Object, "spec")
    sessionType, _ := spec["type"].(string)

    // Default to claude-code for backward compatibility
    if sessionType == "" {
        sessionType = "claude-code"
    }

    switch sessionType {
    case "langflow":
        return HandleLangFlowSession(obj)
    case "claude-code":
        return HandleClaudeCodeSession(obj)  // Existing handler
    default:
        return fmt.Errorf("unsupported session type: %s", sessionType)
    }
}
```

### 4. LangFlow Runner

**New Component:** `components/runners/langflow-runner/`

**File:** `components/runners/langflow-runner/run.py`

```python
#!/usr/bin/env python3
"""
LangFlow Runner for Ambient Code Platform

Executes a LangFlow flow and updates the AgenticSession status.
"""

import os
import sys
import json
import requests
from kubernetes import client, config

def main():
    # Load configuration from environment
    flow_id = os.environ['FLOW_ID']
    flow_input = json.loads(os.environ['FLOW_INPUT'])
    langflow_url = os.environ['LANGFLOW_URL']
    langflow_api_key = os.environ['LANGFLOW_API_KEY']
    session_name = os.environ['SESSION_NAME']
    namespace = os.environ['NAMESPACE']

    print(f"Executing LangFlow flow: {flow_id}")

    # Call LangFlow API
    try:
        response = requests.post(
            f"{langflow_url}/api/v1/run/{flow_id}",
            headers={
                "x-api-key": langflow_api_key,
                "Content-Type": "application/json"
            },
            json={
                "inputs": flow_input,
                "output_type": "chat"
            },
            timeout=3600  # 1 hour timeout
        )
        response.raise_for_status()

        result = response.json()
        print(f"Flow execution completed: {result.get('session_id')}")

        # Update AgenticSession status
        update_session_status(namespace, session_name, {
            "phase": "Completed",
            "flowExecutionId": result.get("session_id"),
            "results": {
                "outputs": result.get("outputs", [])
            }
        })

        return 0

    except requests.exceptions.RequestException as e:
        print(f"Flow execution failed: {e}", file=sys.stderr)

        # Update status with error
        update_session_status(namespace, session_name, {
            "phase": "Failed",
            "message": str(e)
        })

        return 1

def update_session_status(namespace, name, updates):
    """Update AgenticSession status using Kubernetes API"""
    config.load_incluster_config()

    api = client.CustomObjectsApi()

    try:
        # Get current session
        session = api.get_namespaced_custom_object(
            group="vteam.ambient-code",
            version="v1alpha1",
            namespace=namespace,
            plural="agenticsessions",
            name=name
        )

        # Update status
        if "status" not in session:
            session["status"] = {}

        session["status"].update(updates)

        # Patch status subresource
        api.patch_namespaced_custom_object_status(
            group="vteam.ambient-code",
            version="v1alpha1",
            namespace=namespace,
            plural="agenticsessions",
            name=name,
            body=session
        )

        print(f"Updated session status: {updates}")

    except Exception as e:
        print(f"Failed to update session status: {e}", file=sys.stderr)

if __name__ == "__main__":
    sys.exit(main())
```

**File:** `components/runners/langflow-runner/Dockerfile`

```dockerfile
FROM python:3.11-slim

WORKDIR /app

# Install dependencies
RUN pip install --no-cache-dir \
    requests==2.31.0 \
    kubernetes==28.1.0

# Copy runner script
COPY run.py /app/run.py
RUN chmod +x /app/run.py

# Run as non-root
USER 1000

ENTRYPOINT ["python3", "/app/run.py"]
```

**File:** `components/runners/langflow-runner/Makefile`

```makefile
IMAGE_NAME ?= langflow_runner
REGISTRY ?= quay.io/ambient_code
TAG ?= latest

.PHONY: build
build:
	docker build -t $(REGISTRY)/$(IMAGE_NAME):$(TAG) .

.PHONY: push
push: build
	docker push $(REGISTRY)/$(IMAGE_NAME):$(TAG)
```

### 5. LangFlow Deployment

**File:** `components/manifests/langflow/deployment.yaml`

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: ambient-code
---
apiVersion: v1
kind: Secret
metadata:
  name: langflow-secret
  namespace: ambient-code
type: Opaque
stringData:
  api-key: "your-api-key-here"  # Generated on first run or set manually
---
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: langflow-data
  namespace: ambient-code
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 10Gi
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: langflow
  namespace: ambient-code
  labels:
    app: langflow
spec:
  replicas: 1
  selector:
    matchLabels:
      app: langflow
  template:
    metadata:
      labels:
        app: langflow
    spec:
      containers:
      - name: langflow
        image: langflowai/langflow:latest
        ports:
        - containerPort: 7860
          name: http
        env:
        - name: LANGFLOW_DATABASE_URL
          value: "sqlite:////data/langflow.db"
        - name: LANGFLOW_AUTO_LOGIN
          value: "false"
        - name: LANGFLOW_STORE_ENVIRONMENT_VARIABLES
          value: "true"
        volumeMounts:
        - name: data
          mountPath: /data
        resources:
          requests:
            memory: "512Mi"
            cpu: "500m"
          limits:
            memory: "2Gi"
            cpu: "2000m"
        livenessProbe:
          httpGet:
            path: /health
            port: 7860
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /health
            port: 7860
          initialDelaySeconds: 10
          periodSeconds: 5
      volumes:
      - name: data
        persistentVolumeClaim:
          claimName: langflow-data
---
apiVersion: v1
kind: Service
metadata:
  name: langflow
  namespace: ambient-code
  labels:
    app: langflow
spec:
  type: ClusterIP
  ports:
  - port: 7860
    targetPort: 7860
    protocol: TCP
    name: http
  selector:
    app: langflow
---
# OpenShift Route for LangFlow UI access
apiVersion: route.openshift.io/v1
kind: Route
metadata:
  name: langflow
  namespace: ambient-code
spec:
  to:
    kind: Service
    name: langflow
  port:
    targetPort: http
  tls:
    termination: edge
    insecureEdgeTerminationPolicy: Redirect
```

**File:** `components/manifests/langflow/README.md`

```markdown
# LangFlow Deployment

## Installation

```bash
# Deploy LangFlow
oc apply -f components/manifests/langflow/

# Wait for deployment
oc wait --for=condition=available deployment/langflow -n ambient-code --timeout=300s

# Get LangFlow URL
oc get route langflow -n ambient-code -o jsonpath='{.spec.host}'
```

## Configuration

### API Key Setup

1. Access LangFlow UI via the route
2. Create an account or login
3. Go to Settings → API Keys
4. Generate a new API key
5. Update the secret:

```bash
oc create secret generic langflow-secret \
  --from-literal=api-key=YOUR_API_KEY \
  -n ambient-code \
  --dry-run=client -o yaml | oc apply -f -
```

### Backend Configuration

Update backend deployment with LangFlow URL:

```bash
oc set env deployment/backend-api \
  LANGFLOW_URL=http://langflow.ambient-code.svc.cluster.local:7860 \
  -n ambient-code
```

## Verification

```bash
# Check LangFlow is running
curl http://langflow.ambient-code.svc.cluster.local:7860/health

# List flows (requires API key)
curl -H "x-api-key: YOUR_API_KEY" \
  http://langflow.ambient-code.svc.cluster.local:7860/api/v1/flows
```
```

### 6. Frontend Changes

#### Type Definitions

**File:** `components/frontend/src/types/session.ts`

```typescript
export type SessionType = 'claude-code' | 'langflow';

export type LangFlowInput = Record<string, unknown>;

export type AgenticSessionSpec = {
  // Existing fields
  prompt?: string;
  repos?: Repository[];
  interactive?: boolean;
  timeout?: number;
  model?: string;

  // NEW: Type and LangFlow fields
  type?: SessionType;
  flowId?: string;
  flowInput?: LangFlowInput;
};

export type LangFlowFlow = {
  id: string;
  name: string;
  description: string;
  data: Record<string, unknown>;
  updated_at: string;
};
```

#### LangFlow API Service

**File:** `components/frontend/src/services/api/langflow.ts`

```typescript
import { apiClient } from './client';
import type { LangFlowFlow } from '@/types/session';

export const langflowApi = {
  async listFlows(): Promise<LangFlowFlow[]> {
    const response = await apiClient.get('/api/langflow/flows');
    return response.data.flows;
  },

  async getFlow(id: string): Promise<LangFlowFlow> {
    const response = await apiClient.get(`/api/langflow/flows/${id}`);
    return response.data;
  },

  async checkHealth(): Promise<boolean> {
    try {
      await apiClient.get('/api/langflow/health');
      return true;
    } catch {
      return false;
    }
  },
};
```

#### React Query Hooks

**File:** `components/frontend/src/services/queries/langflow.ts`

```typescript
import { useQuery } from '@tanstack/react-query';
import { langflowApi } from '../api/langflow';

export const langflowKeys = {
  all: ['langflow'] as const,
  flows: () => [...langflowKeys.all, 'flows'] as const,
  flow: (id: string) => [...langflowKeys.all, 'flow', id] as const,
  health: () => [...langflowKeys.all, 'health'] as const,
};

export function useLangFlowFlows() {
  return useQuery({
    queryKey: langflowKeys.flows(),
    queryFn: langflowApi.listFlows,
  });
}

export function useLangFlowFlow(id: string) {
  return useQuery({
    queryKey: langflowKeys.flow(id),
    queryFn: () => langflowApi.getFlow(id),
    enabled: !!id,
  });
}

export function useLangFlowHealth() {
  return useQuery({
    queryKey: langflowKeys.health(),
    queryFn: langflowApi.checkHealth,
    refetchInterval: 30000, // Check every 30s
  });
}
```

#### Session Creation Form Component

**File:** `components/frontend/src/app/projects/[name]/sessions/new/components/session-type-selector.tsx`

```typescript
'use client';

import { useState } from 'react';
import { Label } from '@/components/ui/label';
import { RadioGroup, RadioGroupItem } from '@/components/ui/radio-group';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { AlertCircle, CheckCircle2 } from 'lucide-react';
import { useLangFlowHealth } from '@/services/queries/langflow';
import type { SessionType } from '@/types/session';

type SessionTypeSelectorProps = {
  value: SessionType;
  onChange: (type: SessionType) => void;
};

export function SessionTypeSelector({ value, onChange }: SessionTypeSelectorProps) {
  const { data: isLangFlowHealthy, isLoading } = useLangFlowHealth();

  return (
    <div className="space-y-4">
      <Label>Agent Type</Label>
      <RadioGroup value={value} onValueChange={(v) => onChange(v as SessionType)}>
        <div className="flex items-center space-x-2">
          <RadioGroupItem value="claude-code" id="claude-code" />
          <Label htmlFor="claude-code" className="font-normal cursor-pointer">
            Claude Code Agent (Markdown)
          </Label>
        </div>
        <div className="flex items-center space-x-2">
          <RadioGroupItem
            value="langflow"
            id="langflow"
            disabled={!isLangFlowHealthy}
          />
          <Label
            htmlFor="langflow"
            className={`font-normal ${isLangFlowHealthy ? 'cursor-pointer' : 'cursor-not-allowed opacity-50'}`}
          >
            LangFlow Visual Workflow
          </Label>
          {!isLoading && (
            isLangFlowHealthy ? (
              <CheckCircle2 className="h-4 w-4 text-green-500" />
            ) : (
              <AlertCircle className="h-4 w-4 text-yellow-500" />
            )
          )}
        </div>
      </RadioGroup>

      {!isLangFlowHealthy && !isLoading && (
        <Alert variant="destructive">
          <AlertCircle className="h-4 w-4" />
          <AlertDescription>
            LangFlow service is not available. Please contact your administrator.
          </AlertDescription>
        </Alert>
      )}
    </div>
  );
}
```

**File:** `components/frontend/src/app/projects/[name]/sessions/new/components/langflow-selector.tsx`

```typescript
'use client';

import { useLangFlowFlows, useLangFlowFlow } from '@/services/queries/langflow';
import { Label } from '@/components/ui/label';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Skeleton } from '@/components/ui/skeleton';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';

type LangFlowSelectorProps = {
  flowId: string;
  onChange: (flowId: string) => void;
};

export function LangFlowSelector({ flowId, onChange }: LangFlowSelectorProps) {
  const { data: flows, isLoading } = useLangFlowFlows();
  const { data: selectedFlow } = useLangFlowFlow(flowId);

  if (isLoading) {
    return <Skeleton className="h-32 w-full" />;
  }

  return (
    <div className="space-y-4">
      <div>
        <Label htmlFor="flow-select">Select Flow</Label>
        <Select value={flowId} onValueChange={onChange}>
          <SelectTrigger id="flow-select">
            <SelectValue placeholder="Choose a LangFlow workflow..." />
          </SelectTrigger>
          <SelectContent>
            {flows?.map((flow) => (
              <SelectItem key={flow.id} value={flow.id}>
                {flow.name}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      {selectedFlow && (
        <Card>
          <CardHeader>
            <CardTitle className="text-sm">{selectedFlow.name}</CardTitle>
            <CardDescription>{selectedFlow.description}</CardDescription>
          </CardHeader>
          <CardContent>
            <p className="text-sm text-muted-foreground">
              Last updated: {new Date(selectedFlow.updated_at).toLocaleString()}
            </p>
          </CardContent>
        </Card>
      )}
    </div>
  );
}
```

**File:** `components/frontend/src/app/projects/[name]/sessions/new/page.tsx` (updates)

```typescript
'use client';

import { useState } from 'react';
import { SessionTypeSelector } from './components/session-type-selector';
import { LangFlowSelector } from './components/langflow-selector';
import type { SessionType, LangFlowInput } from '@/types/session';

export default function NewSessionPage() {
  const [sessionType, setSessionType] = useState<SessionType>('claude-code');
  const [flowId, setFlowId] = useState<string>('');
  const [flowInput, setFlowInput] = useState<LangFlowInput>({});

  // ... rest of component

  return (
    <div className="space-y-6">
      <SessionTypeSelector
        value={sessionType}
        onChange={setSessionType}
      />

      {sessionType === 'langflow' ? (
        <>
          <LangFlowSelector
            flowId={flowId}
            onChange={setFlowId}
          />
          {/* TODO: Add flow input configuration UI */}
        </>
      ) : (
        <>
          {/* Existing Claude Code form fields */}
        </>
      )}

      {/* Submit button, etc. */}
    </div>
  );
}
```

### 7. Configuration Management

**Backend Environment Variables:**

Add to `components/backend/main.go`:

```go
langflowURL := os.Getenv("LANGFLOW_URL")
if langflowURL == "" {
    langflowURL = "http://langflow.ambient-code.svc.cluster.local:7860"
}

langflowAPIKey := os.Getenv("LANGFLOW_API_KEY")
if langflowAPIKey == "" {
    log.Println("Warning: LANGFLOW_API_KEY not set")
}

langflowClient = langflow.NewClient(langflowURL, langflowAPIKey)
```

**Backend Deployment Update:**

```yaml
# components/manifests/backend-deployment.yaml
env:
- name: LANGFLOW_URL
  value: "http://langflow.ambient-code.svc.cluster.local:7860"
- name: LANGFLOW_API_KEY
  valueFrom:
    secretKeyRef:
      name: langflow-secret
      key: api-key
```

## Implementation Plan

### Phase 1: Foundation (Week 1)

**Goals:** Deploy LangFlow, establish connectivity

**Tasks:**
1. Deploy LangFlow to cluster
   - Apply manifests
   - Verify service is accessible
   - Configure API key
2. Create langflow-runner component
   - Implement Python script
   - Build and push container image
   - Test manual execution
3. Update backend with LangFlow client
   - Implement client.go
   - Add configuration
   - Test API connectivity

**Success Criteria:**
- ✅ LangFlow UI accessible via route
- ✅ Backend can list flows from LangFlow API
- ✅ langflow-runner can execute a test flow

### Phase 2: Backend Integration (Week 2)

**Goals:** AgenticSession support, operator routing

**Tasks:**
1. Extend AgenticSession CRD
   - Add `type`, `flowId`, `flowInput` fields
   - Update CRD manifests
   - Apply to cluster
2. Implement backend handlers
   - `GET /api/langflow/flows`
   - `GET /api/langflow/flows/:id`
   - `GET /api/langflow/health`
3. Update session creation handler
   - Validate LangFlow session specs
   - Store flowId and flowInput in CR
4. Operator session routing
   - Detect session type
   - Route to appropriate handler
   - Create langflow-runner jobs

**Success Criteria:**
- ✅ Can create LangFlow-type sessions via API
- ✅ Operator spawns langflow-runner jobs
- ✅ Jobs execute and update session status

### Phase 3: Frontend UI (Week 3)

**Goals:** User-facing flow selection interface

**Tasks:**
1. Implement type definitions
   - Add SessionType, LangFlowFlow types
2. Create API services and query hooks
   - langflowApi service
   - React Query hooks
3. Build session creation UI
   - SessionTypeSelector component
   - LangFlowSelector component
   - Flow input configuration (basic)
4. Update session list/detail views
   - Display flow name for LangFlow sessions
   - Show flow execution results

**Success Criteria:**
- ✅ Users can select between Claude Code and LangFlow
- ✅ Users can browse and select LangFlow flows
- ✅ Sessions execute and display results

### Phase 4: Polish & Documentation (Week 4)

**Goals:** Production readiness, documentation

**Tasks:**
1. Error handling improvements
   - LangFlow service unavailable handling
   - Flow execution timeout handling
   - Better error messages
2. Monitoring and observability
   - Add metrics for LangFlow sessions
   - Log aggregation
3. Documentation
   - User guide for creating LangFlow flows
   - Admin guide for LangFlow deployment
   - API documentation updates
4. Testing
   - E2E tests for LangFlow sessions
   - Integration tests

**Success Criteria:**
- ✅ Comprehensive error handling
- ✅ Documentation complete
- ✅ Tests passing

### Phase 5: Advanced Features (Future)

**Potential Enhancements:**

1. **Flow Input UI Builder**
   - Dynamic form generation based on flow inputs
   - Type validation
   - Default value support

2. **Flow Sync Automation**
   - Webhook from LangFlow on flow save
   - Automatic flow discovery
   - Version tracking

3. **Embedded Flow Editor**
   - iframe LangFlow UI in Ambient Code
   - Single sign-on integration
   - Seamless workflow

4. **Flow Templates**
   - Pre-built flows for common tasks
   - Flow marketplace
   - Import/export

5. **Multi-Environment Support**
   - Dev/staging/prod LangFlow instances
   - Environment-specific flows

## Testing Strategy

### Unit Tests

**Backend:**
- LangFlow client methods
- Session validation logic
- Type routing

**Operator:**
- Job creation for LangFlow sessions
- Status update logic

### Integration Tests

**Flow Execution:**
```bash
# Create a test flow in LangFlow
# Create AgenticSession via API
# Verify job created
# Verify flow executed
# Verify status updated
```

**API Endpoints:**
```bash
# Test GET /api/langflow/flows
# Test GET /api/langflow/flows/:id
# Test GET /api/langflow/health
```

### E2E Tests (Cypress)

```typescript
describe('LangFlow Integration', () => {
  it('should create a LangFlow session', () => {
    cy.visit('/projects/test-project/sessions/new');

    // Select LangFlow type
    cy.get('[data-testid="session-type-langflow"]').click();

    // Select flow
    cy.get('[data-testid="flow-select"]').click();
    cy.contains('My Test Flow').click();

    // Submit
    cy.get('[data-testid="create-session-btn"]').click();

    // Verify redirect
    cy.url().should('include', '/sessions/');

    // Verify execution
    cy.contains('Running').should('be.visible');
  });
});
```

## Security Considerations

### API Key Management

- LangFlow API key stored in Kubernetes Secret
- Never logged or exposed to users
- Rotatable without code changes

### RBAC

- LangFlow service accessible only within cluster (ClusterIP)
- Runner pods have minimal permissions (can only update owning AgenticSession)
- Users cannot directly access LangFlow API (proxied through backend)

### Network Policies

```yaml
# Optional: Restrict LangFlow access
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: langflow-access
  namespace: ambient-code
spec:
  podSelector:
    matchLabels:
      app: langflow
  ingress:
  - from:
    - podSelector:
        matchLabels:
          app: backend-api
    - podSelector:
        matchLabels:
          app: langflow-runner
```

## Migration & Rollback

### Backward Compatibility

- Existing AgenticSessions continue to work (default type: `claude-code`)
- No changes required to existing workflows
- LangFlow is additive, not replacing existing functionality

### Rollback Plan

If issues arise:

1. Remove LangFlow route registration from backend
2. Scale LangFlow deployment to 0 replicas
3. Existing sessions unaffected
4. Can redeploy when ready

### Data Migration

Not applicable - no existing data to migrate. This is a new capability.

## Monitoring & Observability

### Metrics to Track

1. **LangFlow Session Execution**
   - Total sessions created (by type)
   - Success/failure rate
   - Execution duration
   - Flow popularity (which flows used most)

2. **LangFlow Service Health**
   - API availability
   - Response times
   - Error rates

3. **Resource Usage**
   - LangFlow pod CPU/memory
   - Runner pod resource consumption

### Logging

**Structured Logs:**
```json
{
  "timestamp": "2025-01-10T12:00:00Z",
  "level": "info",
  "session": "my-session",
  "type": "langflow",
  "flowId": "abc-123",
  "event": "execution_started"
}
```

**Key Events to Log:**
- Flow selection
- Execution start/complete
- Errors and retries
- Status updates

## Open Questions & Future Considerations

### Questions to Resolve

1. **Flow Input Configuration**
   - How to build dynamic UI for flow inputs?
   - Should we parse flow JSON to determine input schema?
   - Or require manual input configuration?

2. **Flow Discovery**
   - Polling interval for flow sync?
   - Webhook integration feasible?
   - Cache strategy?

3. **Multi-Tenancy**
   - Should each project have isolated LangFlow instance?
   - Or shared LangFlow with namespace isolation?

4. **Authentication**
   - Single shared API key, or per-user/per-project keys?
   - SSO integration with LangFlow?

### Future Enhancements

1. **Flow Versioning**
   - Track flow versions
   - Pin sessions to specific versions
   - Version comparison

2. **Flow Marketplace**
   - Community-contributed flows
   - Flow templates
   - Import from external sources

3. **Hybrid Workflows**
   - Combine Claude Code agents with LangFlow flows
   - Multi-step workflows with mixed types

4. **Advanced Monitoring**
   - LangFlow execution traces
   - Performance profiling
   - Cost tracking (if using paid LLM APIs)

## Success Metrics

### Phase 1-2 (Foundation)
- LangFlow deployed and operational
- Backend can communicate with LangFlow API
- At least one test session executes successfully

### Phase 3-4 (User-Facing)
- 5+ users create LangFlow sessions
- 80%+ session success rate
- < 5 second flow selection UI load time

### Long-Term (6 months)
- 30%+ of sessions use LangFlow
- 20+ flows created by users
- Zero production incidents related to LangFlow integration

## References

- [LangFlow GitHub Repository](https://github.com/langflow-ai/langflow)
- [LangFlow Documentation](https://docs.langflow.org)
- [LangFlow API Reference](https://docs.langflow.org/api-reference-api-examples)
- Ambient Code Platform Architecture (see `CLAUDE.md`)
- AgenticSession CRD Specification

## Change Log

| Date | Version | Author | Changes |
|------|---------|--------|---------|
| 2025-01-10 | 1.0 | Claude Code | Initial design document |
