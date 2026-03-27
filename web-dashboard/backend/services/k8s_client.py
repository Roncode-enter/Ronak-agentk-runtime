"""Kubernetes API client wrapper.

Talks to the cluster using the uploaded kubeconfig or in-cluster config.
All agent/cost/attestation data lives in Kubernetes — no database needed.
"""
import tempfile
import os
from typing import Optional
from kubernetes import client, config

_api_client: Optional[client.ApiClient] = None
_kubeconfig_path: Optional[str] = None


def set_kubeconfig_data(kubeconfig_content: str):
    """Load kubeconfig from uploaded content."""
    global _api_client, _kubeconfig_path
    fd, path = tempfile.mkstemp(suffix=".yaml")
    with os.fdopen(fd, "w") as f:
        f.write(kubeconfig_content)
    _kubeconfig_path = path
    _api_client = None  # Force re-init
    _get_client()  # Validate it works


def _get_client() -> client.ApiClient:
    global _api_client
    if _api_client is not None:
        return _api_client

    try:
        if _kubeconfig_path:
            config.load_kube_config(config_file=_kubeconfig_path)
        elif os.getenv("KUBECONFIG"):
            config.load_kube_config(config_file=os.getenv("KUBECONFIG"))
        else:
            config.load_incluster_config()
        _api_client = client.ApiClient()
        return _api_client
    except Exception:
        raise RuntimeError("No Kubernetes cluster connected. Upload a kubeconfig first.")


def get_custom_api() -> client.CustomObjectsApi:
    return client.CustomObjectsApi(_get_client())


def get_apps_api() -> client.AppsV1Api:
    return client.AppsV1Api(_get_client())


GROUP = "runtime.agentic-layer.ai"
VERSION = "v1alpha1"


def list_agents(namespace: str = "default") -> list:
    """List all Agent CRs."""
    api = get_custom_api()
    result = api.list_namespaced_custom_object(GROUP, VERSION, namespace, "agents")
    return result.get("items", [])


def get_agent(name: str, namespace: str = "default") -> dict:
    """Get a single Agent CR."""
    api = get_custom_api()
    return api.get_namespaced_custom_object(GROUP, VERSION, namespace, "agents", name)


def create_agent(body: dict, namespace: str = "default") -> dict:
    """Create an Agent CR from YAML dict."""
    api = get_custom_api()
    return api.create_namespaced_custom_object(GROUP, VERSION, namespace, "agents", body)


def delete_agent(name: str, namespace: str = "default"):
    """Delete an Agent CR."""
    api = get_custom_api()
    return api.delete_namespaced_custom_object(GROUP, VERSION, namespace, "agents", name)


def list_confidential_agents(namespace: str = "default") -> list:
    api = get_custom_api()
    result = api.list_namespaced_custom_object(GROUP, VERSION, namespace, "confidentialagents")
    return result.get("items", [])


def get_confidential_agent(name: str, namespace: str = "default") -> dict:
    api = get_custom_api()
    return api.get_namespaced_custom_object(GROUP, VERSION, namespace, "confidentialagents", name)


def list_simulation_previews(namespace: str = "default") -> list:
    api = get_custom_api()
    result = api.list_namespaced_custom_object(GROUP, VERSION, namespace, "simulationpreviews")
    return result.get("items", [])
