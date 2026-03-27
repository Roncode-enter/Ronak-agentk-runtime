"""Agent CRUD router — list, create, delete agents via Kubernetes API."""
from fastapi import APIRouter, HTTPException, Depends
from pydantic import BaseModel
from typing import Optional

from backend.routers.auth import get_current_user
from backend.services import k8s_client
from backend.services.yaml_generator import generate_agent_yaml

router = APIRouter()


class CreateAgentRequest(BaseModel):
    name: str
    description: str = ""
    instruction: str = ""
    framework: str = "google-adk"
    image: Optional[str] = None
    model: Optional[str] = "gemini/gemini-2.5-flash"
    replicas: int = 1
    apiKey: Optional[str] = None
    maxMonthlyCost: Optional[float] = None
    costPerToken: Optional[str] = "0.00001"
    downgradeModel: Optional[str] = None
    optimizationMode: Optional[str] = None
    spotFallback: bool = False
    suspendOnExhaust: bool = False
    verifiableEnabled: bool = False
    proofMode: str = "snark-groth16"
    autonomyLevel: Optional[int] = None
    requireCompliance: bool = True
    humanWebhook: Optional[str] = None
    strategy: Optional[str] = None
    selfHealing: bool = True
    promptVersion: Optional[str] = None
    namespace: str = "default"


@router.get("")
def list_agents(namespace: str = "default", user: dict = Depends(get_current_user)):
    """List all agents with their status."""
    try:
        agents = k8s_client.list_agents(namespace)
        result = []
        for a in agents:
            status = a.get("status", {})
            spec = a.get("spec", {})
            result.append({
                "name": a["metadata"]["name"],
                "namespace": a["metadata"]["namespace"],
                "framework": spec.get("framework", ""),
                "replicas": spec.get("replicas", 1),
                "merkleRoot": status.get("merkleRoot", ""),
                "zkProofRoot": status.get("zkProofRoot", ""),
                "attestationDigest": status.get("attestationDigest", ""),
                "predictedMonthlyCostUSD": status.get("predictedMonthlyCostUSD", ""),
                "costAction": status.get("costAction", ""),
                "governanceStatus": status.get("governanceStatus", ""),
                "lifecyclePhase": status.get("lifecyclePhase", ""),
                "tokensUsed": status.get("tokensUsed", 0),
                "proofMode": spec.get("verifiable", {}).get("proofMode", ""),
                "conditions": status.get("conditions", []),
            })
        return {"agents": result}
    except RuntimeError as e:
        raise HTTPException(status_code=503, detail=str(e))
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))


@router.post("")
def create_agent(req: CreateAgentRequest, user: dict = Depends(get_current_user)):
    """Create a new agent from form data."""
    # Check premium for PlonK
    if req.proofMode == "plonk-universal" and not user.get("premium"):
        raise HTTPException(status_code=403, detail="PlonK proofs require a Premium subscription ($49/month)")

    try:
        body = generate_agent_yaml(req.model_dump())
        result = k8s_client.create_agent(body, req.namespace)
        return {"status": "created", "name": req.name}
    except Exception as e:
        raise HTTPException(status_code=400, detail=str(e))


@router.get("/{name}")
def get_agent(name: str, namespace: str = "default", user: dict = Depends(get_current_user)):
    try:
        return k8s_client.get_agent(name, namespace)
    except Exception as e:
        raise HTTPException(status_code=404, detail=str(e))


@router.delete("/{name}")
def delete_agent(name: str, namespace: str = "default", user: dict = Depends(get_current_user)):
    try:
        k8s_client.delete_agent(name, namespace)
        return {"status": "deleted", "name": name}
    except Exception as e:
        raise HTTPException(status_code=400, detail=str(e))
