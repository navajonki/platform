# LangFlow Integration - Implementation Status

**Last Updated:** 2025-01-11
**Branch:** `feature/langflow-integration`
**Fork:** https://github.com/navajonki/platform

## Quick Summary

✅ **Phase 1 (Foundation): COMPLETED**
🚧 **Phase 2 (Backend Integration): READY TO START**
📋 **Phase 3 (Frontend UI): PENDING**
📋 **Phase 4 (Polish & Documentation): PENDING**

---

## Phase 1: Foundation ✅ COMPLETED

### What Was Built

**1. LangFlow Deployment**
- Deployed to Minikube with SQLite backend and persistent storage
- Created Kubernetes manifests (Deployment, Service, Ingress, Secret, PVC)
- Fixed PORT environment variable conflict (LANGFLOW_PORT override)
- Status: **Running and accessible at http://langflow.local**

**2. LangFlow Runner Component**
- Python-based runner for executing LangFlow workflows
- Kubernetes API integration for AgenticSession status updates
- Comprehensive error handling with 1-hour timeout
- Docker image built and loaded into Minikube
- Location: `components/runners/langflow-runner/`

**3. Backend API Integration**
- Go client library for LangFlow REST API (`components/backend/langflow/client.go`)
- Supports: health checks, list flows, get flow, run flow, download flow
- Three new API endpoints:
  - `GET /api/langflow/health` - Check LangFlow availability
  - `GET /api/langflow/flows` - List all flows
  - `GET /api/langflow/flows/:id` - Get specific flow
- Client initialized in `main.go` with environment variables

**4. Comprehensive Testing**
- **30 unit tests** covering all client methods
- **8 contract tests** for HTTP handlers
- **Edge cases tested:**
  - Network failures, timeouts (30s validation)
  - Malformed JSON, empty responses
  - HTTP error codes (401, 403, 500, 502)
  - Special characters in IDs
  - Large payloads, nil inputs, binary data
  - Concurrent requests (10 parallel)
- **All tests passing** (0 failures)

### Files Created

```
components/manifests/langflow/
├── deployment.yaml       # LangFlow service deployment
├── ingress.yaml         # Ingress for UI access
└── README.md            # Setup and troubleshooting guide

components/runners/langflow-runner/
├── run.py               # Python runner script
├── Dockerfile           # Container image
├── Makefile            # Build automation
└── README.md           # Runner documentation

components/backend/langflow/
└── client.go           # LangFlow API client

components/backend/handlers/
└── langflow.go         # HTTP handlers for endpoints

components/backend/tests/unit/langflow/
├── client_test.go                      # Basic unit tests
├── client_edge_cases_test.go          # Edge case tests
└── client_intentional_failures_test.go # Contract validation

components/backend/tests/contract/langflow/
└── handlers_test.go    # Handler contract tests
```

### Files Modified

- `components/backend/routes.go` - Added LangFlow endpoints
- `components/backend/main.go` - Client initialization
- `docs/design/langflow-integration.md` - Updated with status

### Test Results

```bash
# Run all Phase 1 tests
cd components/backend
podman run --rm -v $(pwd):/workspace -w /workspace golang:1.24 \
  go test ./tests/unit/langflow/... ./tests/contract/langflow/...

# Results:
# ok  	ambient-code-backend/tests/unit/langflow	35.543s
# ok  	ambient-code-backend/tests/contract/langflow	0.008s
# Total: 30 tests, 0 failures
```

---

## Phase 2: Backend Integration 🚧 NEXT

### Tasks to Complete

#### 1. Extend AgenticSession CRD

**Location:** `components/manifests/base/crds/agenticsession-crd.yaml`

**Changes Needed:**
```yaml
spec:
  # NEW: Add type field
  type:
    type: string
    enum: ["claude-code", "langflow"]
    default: "claude-code"

  # NEW: LangFlow-specific fields
  flowId:
    type: string
    description: "LangFlow flow ID (required when type=langflow)"

  flowInput:
    type: object
    additionalProperties: true
    description: "Input parameters for the flow"

status:
  # NEW: LangFlow execution tracking
  flowExecutionId:
    type: string
    description: "LangFlow execution session ID"
```

**Testing:**
- Validate CRD accepts new fields
- Test backward compatibility (existing sessions still work)
- Validate enum constraint on `type` field

#### 2. Update Session Creation Handler

**Location:** `components/backend/handlers/sessions.go`

**Changes in `CreateAgenticSession` function:**
```go
// After parsing request body, add validation
sessionType := spec["type"].(string)
if sessionType == "" {
    sessionType = "claude-code" // Default for backward compatibility
}

if sessionType == "langflow" {
    // Validate flowId is present
    flowID, ok := spec["flowId"].(string)
    if !ok || flowID == "" {
        c.JSON(http.StatusBadRequest, gin.H{
            "error": "flowId required for langflow sessions",
        })
        return
    }

    // Validate flow exists in LangFlow
    if handlers.LangFlowClient != nil {
        _, err := handlers.LangFlowClient.GetFlow(flowID)
        if err != nil {
            c.JSON(http.StatusBadRequest, gin.H{
                "error": fmt.Sprintf("Invalid flowId: %v", err),
            })
            return
        }
    }
}
```

**Testing:**
- Test creating claude-code session (existing behavior)
- Test creating langflow session with valid flowId
- Test error when flowId missing
- Test error when flowId doesn't exist in LangFlow

#### 3. Operator Session Routing

**Location:** `components/operator/internal/handlers/sessions.go`

**Add type detection at top of handler:**
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
        return HandleClaudeCodeSession(obj) // Existing handler (rename current function)
    default:
        return fmt.Errorf("unsupported session type: %s", sessionType)
    }
}
```

**Testing:**
- Test routing to claude-code handler (default)
- Test routing to langflow handler
- Test error for unknown type

#### 4. LangFlow Session Handler

**Location:** `components/operator/internal/handlers/langflow_sessions.go` (NEW FILE)

**Implementation:**
```go
func HandleLangFlowSession(obj *unstructured.Unstructured) error {
    name := obj.GetName()
    namespace := obj.GetNamespace()

    // Extract flowId and flowInput from spec
    spec, _, _ := unstructured.NestedMap(obj.Object, "spec")
    flowID := spec["flowId"].(string)
    flowInput := spec["flowInput"].(map[string]interface{})

    // Create Job with langflow-runner
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

    // Start monitoring
    go monitorLangFlowJob(job.Name, name, namespace)

    return nil
}
```

**Job Creation:**
```go
func createLangFlowJob(namespace, sessionName, flowID string,
                       flowInput map[string]interface{},
                       owner *unstructured.Unstructured) *batchv1.Job {

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
                            Image: "vteam-langflow-runner:latest",
                            ImagePullPolicy: corev1.PullNever,
                            Env: []corev1.EnvVar{
                                {Name: "FLOW_ID", Value: flowID},
                                {Name: "FLOW_INPUT", Value: string(inputJSON)},
                                {Name: "LANGFLOW_URL", Value: "http://langflow.ambient-code.svc.cluster.local:7860"},
                                {Name: "LANGFLOW_API_KEY", ValueFrom: &corev1.EnvVarSource{
                                    SecretKeyRef: &corev1.SecretKeySelector{
                                        LocalObjectReference: corev1.LocalObjectReference{
                                            Name: "langflow-secret",
                                        },
                                        Key: "api-key",
                                    },
                                }},
                                {Name: "SESSION_NAME", Value: sessionName},
                                {Name: "NAMESPACE", Value: namespace},
                            },
                        },
                    },
                },
            },
        },
    }
}
```

**Testing:**
- Integration test: Create langflow session CR, verify Job created
- Verify environment variables set correctly
- Verify secret mounted
- Test job monitoring updates session status

#### 5. Phase 2 Testing

**New test files to create:**
- `components/backend/tests/unit/handlers/session_validation_test.go` - CRD validation
- `components/operator/tests/unit/routing_test.go` - Session type routing
- `components/operator/tests/integration/langflow_session_test.go` - End-to-end flow

---

## Phase 3: Frontend UI 📋 PENDING

### Tasks

1. **Type Definitions** (`components/frontend/src/types/session.ts`)
2. **LangFlow API Service** (`src/services/api/langflow.ts`)
3. **React Query Hooks** (`src/services/queries/langflow.ts`)
4. **UI Components:**
   - SessionTypeSelector
   - LangFlowSelector
   - FlowInputForm
5. **Update Session List** - Show "LangFlow" badge
6. **Frontend Tests** - Component tests, query hook tests

---

## Phase 4: Polish & Documentation 📋 PENDING

### Tasks

1. Error handling improvements
2. Monitoring and observability
3. E2E tests (Cypress)
4. User guide documentation
5. Admin guide documentation

---

## How to Resume Work

### Prerequisites

```bash
# Ensure Minikube is running
minikube status

# Ensure LangFlow is running
kubectl get pods -n ambient-code -l app=langflow
# Should show: Running

# Check LangFlow health
curl http://langflow.local/health
# (Or: kubectl port-forward svc/langflow 7860:7860 -n ambient-code)
```

### Start Phase 2

1. **Update CRD:**
   ```bash
   # Edit the CRD
   vim components/manifests/base/crds/agenticsession-crd.yaml

   # Apply changes
   kubectl apply -f components/manifests/base/crds/
   ```

2. **Update Backend Handler:**
   ```bash
   vim components/backend/handlers/sessions.go
   # Add validation logic for type="langflow"
   ```

3. **Update Operator:**
   ```bash
   vim components/operator/internal/handlers/sessions.go
   # Add session type routing

   # Create new file
   vim components/operator/internal/handlers/langflow_sessions.go
   ```

4. **Build and Deploy:**
   ```bash
   # Build backend
   cd components/backend
   podman build -t vteam-backend:v5 .
   podman save localhost/vteam-backend:v5 -o /tmp/backend.tar
   minikube image load /tmp/backend.tar
   kubectl set image deployment/backend-api backend-api=localhost/vteam-backend:v5 -n ambient-code

   # Build operator
   cd components/operator
   podman build -t vteam-operator:v5 .
   podman save localhost/vteam-operator:v5 -o /tmp/operator.tar
   minikube image load /tmp/operator.tar
   kubectl set image deployment/agentic-operator agentic-operator=localhost/vteam-operator:v5 -n ambient-code
   ```

5. **Test End-to-End:**
   ```bash
   # Create a test flow in LangFlow UI
   # Then create a session via API:
   curl -X POST http://backend-url/api/projects/test/agentic-sessions \
     -H "Content-Type: application/json" \
     -d '{
       "name": "test-langflow-session",
       "type": "langflow",
       "flowId": "YOUR_FLOW_ID",
       "flowInput": {"message": "test"}
     }'

   # Watch the session
   kubectl get agenticsession test-langflow-session -n test -w
   ```

---

## Key Design Decisions

1. **Backward Compatibility:** All existing AgenticSessions continue to work (default type="claude-code")
2. **No Export/Conversion:** Flows execute in native LangFlow environment via API
3. **Unified Monitoring:** Both session types use same AgenticSession CR for status tracking
4. **Security:** API keys stored in Kubernetes Secrets, never logged

---

## Success Criteria

**Phase 1:** ✅
- LangFlow deployed and accessible
- Backend can list and retrieve flows
- Comprehensive test coverage (30 tests)

**Phase 2:** (To be validated)
- Can create langflow-type sessions via API
- Operator spawns langflow-runner jobs
- Jobs execute and update session status
- All tests passing

**Phase 3:** (To be validated)
- Users can select LangFlow flows in UI
- Flow inputs configurable
- Sessions display with "LangFlow" badge

**Phase 4:** (To be validated)
- E2E tests passing
- Documentation complete
- Production-ready error handling

---

## Links

- **Fork:** https://github.com/navajonki/platform
- **Branch:** feature/langflow-integration
- **Design Doc:** `docs/design/langflow-integration.md`
- **LangFlow Docs:** https://docs.langflow.org
- **User Flows:** See design doc section "User Flows"
