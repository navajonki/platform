#!/usr/bin/env python3
"""
LangFlow Runner for Ambient Code Platform

Executes a LangFlow flow and updates the AgenticSession status.
"""

import os
import sys
import json
import requests
import time
from kubernetes import client, config
from typing import Dict, Any, Optional


def main() -> int:
    """Main execution function"""
    try:
        # Load configuration from environment
        flow_id = os.environ.get('FLOW_ID')
        flow_input_raw = os.environ.get('FLOW_INPUT', '{}')
        langflow_url = os.environ.get('LANGFLOW_URL')
        langflow_api_key = os.environ.get('LANGFLOW_API_KEY')
        session_name = os.environ.get('SESSION_NAME')
        namespace = os.environ.get('NAMESPACE')

        # Validate required environment variables
        if not all([flow_id, langflow_url, session_name, namespace]):
            print("ERROR: Missing required environment variables", file=sys.stderr)
            print(f"FLOW_ID: {flow_id}", file=sys.stderr)
            print(f"LANGFLOW_URL: {langflow_url}", file=sys.stderr)
            print(f"SESSION_NAME: {session_name}", file=sys.stderr)
            print(f"NAMESPACE: {namespace}", file=sys.stderr)
            return 1

        # Parse flow input
        try:
            flow_input = json.loads(flow_input_raw)
        except json.JSONDecodeError as e:
            print(f"ERROR: Invalid flow input JSON: {e}", file=sys.stderr)
            update_session_status(namespace, session_name, {
                "phase": "Failed",
                "message": f"Invalid flow input JSON: {e}"
            })
            return 1

        print(f"Executing LangFlow flow: {flow_id}")
        print(f"LangFlow URL: {langflow_url}")
        print(f"Session: {namespace}/{session_name}")
        print(f"Flow input: {json.dumps(flow_input, indent=2)}")

        # Update status to Running
        update_session_status(namespace, session_name, {
            "phase": "Running",
            "startTime": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime())
        })

        # Execute the flow
        result = execute_langflow_flow(
            langflow_url=langflow_url,
            api_key=langflow_api_key,
            flow_id=flow_id,
            inputs=flow_input
        )

        if result:
            print(f"Flow execution completed successfully")
            print(f"Session ID: {result.get('session_id', 'N/A')}")

            # Update status to Completed
            update_session_status(namespace, session_name, {
                "phase": "Completed",
                "completionTime": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
                "flowExecutionId": result.get("session_id", ""),
                "results": {
                    "outputs": result.get("outputs", [])
                }
            })

            return 0
        else:
            print("Flow execution failed", file=sys.stderr)
            return 1

    except Exception as e:
        print(f"Unexpected error: {e}", file=sys.stderr)
        import traceback
        traceback.print_exc()

        # Attempt to update status
        try:
            update_session_status(namespace, session_name, {
                "phase": "Failed",
                "message": str(e)
            })
        except Exception as update_err:
            print(f"Failed to update status: {update_err}", file=sys.stderr)

        return 1


def execute_langflow_flow(
    langflow_url: str,
    api_key: Optional[str],
    flow_id: str,
    inputs: Dict[str, Any]
) -> Optional[Dict[str, Any]]:
    """
    Execute a LangFlow flow via API

    Args:
        langflow_url: Base URL of LangFlow instance
        api_key: API key for authentication (optional for local dev)
        flow_id: Flow ID to execute
        inputs: Input parameters for the flow

    Returns:
        Flow execution result or None on failure
    """
    url = f"{langflow_url}/api/v1/run/{flow_id}"

    headers = {
        "Content-Type": "application/json"
    }

    if api_key:
        headers["x-api-key"] = api_key

    payload = {
        "inputs": inputs,
        "output_type": "chat"
    }

    print(f"POST {url}")
    print(f"Headers: {json.dumps({k: v if k != 'x-api-key' else '[REDACTED]' for k, v in headers.items()})}")
    print(f"Payload: {json.dumps(payload, indent=2)}")

    try:
        response = requests.post(
            url,
            headers=headers,
            json=payload,
            timeout=3600  # 1 hour timeout
        )

        print(f"Response status: {response.status_code}")

        if response.status_code == 200:
            result = response.json()
            print(f"Response body: {json.dumps(result, indent=2)}")
            return result
        else:
            print(f"ERROR: Flow execution failed with status {response.status_code}", file=sys.stderr)
            print(f"Response: {response.text}", file=sys.stderr)

            # Update session with error
            namespace = os.environ.get('NAMESPACE')
            session_name = os.environ.get('SESSION_NAME')
            if namespace and session_name:
                update_session_status(namespace, session_name, {
                    "phase": "Failed",
                    "message": f"LangFlow API error {response.status_code}: {response.text[:200]}"
                })

            return None

    except requests.exceptions.Timeout:
        print("ERROR: Flow execution timed out", file=sys.stderr)
        namespace = os.environ.get('NAMESPACE')
        session_name = os.environ.get('SESSION_NAME')
        if namespace and session_name:
            update_session_status(namespace, session_name, {
                "phase": "Failed",
                "message": "Flow execution timed out after 1 hour"
            })
        return None

    except requests.exceptions.RequestException as e:
        print(f"ERROR: Request failed: {e}", file=sys.stderr)
        namespace = os.environ.get('NAMESPACE')
        session_name = os.environ.get('SESSION_NAME')
        if namespace and session_name:
            update_session_status(namespace, session_name, {
                "phase": "Failed",
                "message": f"Request failed: {str(e)}"
            })
        return None


def update_session_status(namespace: str, name: str, updates: Dict[str, Any]) -> None:
    """
    Update AgenticSession status using Kubernetes API

    Args:
        namespace: Namespace of the session
        name: Name of the session
        updates: Dictionary of status fields to update
    """
    try:
        # Load in-cluster config
        config.load_incluster_config()

        api = client.CustomObjectsApi()

        # Get current session
        try:
            session = api.get_namespaced_custom_object(
                group="vteam.ambient-code",
                version="v1alpha1",
                namespace=namespace,
                plural="agenticsessions",
                name=name
            )
        except client.exceptions.ApiException as e:
            if e.status == 404:
                print(f"WARNING: Session {namespace}/{name} not found, cannot update status", file=sys.stderr)
                return
            raise

        # Initialize status if not present
        if "status" not in session:
            session["status"] = {}

        # Apply updates
        for key, value in updates.items():
            session["status"][key] = value

        # Patch status subresource
        api.patch_namespaced_custom_object_status(
            group="vteam.ambient-code",
            version="v1alpha1",
            namespace=namespace,
            plural="agenticsessions",
            name=name,
            body=session
        )

        print(f"Updated session status: {json.dumps(updates, indent=2)}")

    except Exception as e:
        print(f"ERROR: Failed to update session status: {e}", file=sys.stderr)
        import traceback
        traceback.print_exc()


if __name__ == "__main__":
    sys.exit(main())
