"""Attestation router — TEE attestation reports from ConfidentialAgent status."""
from fastapi import APIRouter, HTTPException, Depends
from backend.routers.auth import get_current_user
from backend.services import k8s_client

router = APIRouter()


@router.get("/{name}")
def get_attestation(name: str, namespace: str = "default", user: dict = Depends(get_current_user)):
    """Get attestation report for a ConfidentialAgent."""
    try:
        ca = k8s_client.get_confidential_agent(name, namespace)
        status = ca.get("status", {})
        spec = ca.get("spec", {})

        return {
            "name": name,
            "agentRef": spec.get("agentRef", {}).get("name", ""),
            "teeProvider": status.get("teeProvider", spec.get("provider", "")),
            "verified": status.get("verified", False),
            "attestationReport": status.get("attestationReport", ""),
            "lastAttestationTime": status.get("lastAttestationTime", ""),
            "deploymentName": status.get("deploymentName", ""),
            "runtimeClassName": spec.get("runtimeClassName", "kata-cc"),
            "memoryEncryption": spec.get("memoryEncryption", True),
            "enclaveMemoryMB": spec.get("enclaveMemoryMB", 256),
            "conditions": status.get("conditions", []),
        }
    except Exception as e:
        raise HTTPException(status_code=404, detail=str(e))


@router.get("")
def list_attestations(namespace: str = "default", user: dict = Depends(get_current_user)):
    """List all ConfidentialAgent attestation reports."""
    try:
        cas = k8s_client.list_confidential_agents(namespace)
        results = []
        for ca in cas:
            status = ca.get("status", {})
            spec = ca.get("spec", {})
            results.append({
                "name": ca["metadata"]["name"],
                "teeProvider": status.get("teeProvider", spec.get("provider", "")),
                "verified": status.get("verified", False),
                "agentRef": spec.get("agentRef", {}).get("name", ""),
                "lastAttestationTime": status.get("lastAttestationTime", ""),
            })
        return {"attestations": results}
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))
