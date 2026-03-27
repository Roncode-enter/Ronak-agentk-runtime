"""Simulation router — preview what would happen before deploying an agent."""
from fastapi import APIRouter, HTTPException, Depends
from backend.routers.auth import get_current_user
from backend.services import k8s_client
from backend.services.yaml_generator import generate_agent_yaml
import yaml

router = APIRouter()


@router.post("/preview")
def preview_simulation(form_data: dict, user: dict = Depends(get_current_user)):
    """Generate a dry-run preview of what deploying this agent would create."""
    try:
        agent_yaml = generate_agent_yaml(form_data)
        yaml_str = yaml.dump(agent_yaml, default_flow_style=False)

        # Estimate resources
        replicas = form_data.get("replicas", 1)
        has_tee = form_data.get("verifiableEnabled", False)

        return {
            "yaml": yaml_str,
            "estimatedResources": {
                "pods": replicas,
                "services": 1,
                "containers_per_pod": 2 if has_tee else 1,
            },
            "estimatedMonthlyCost": f"${float(form_data.get('maxMonthlyCost', 0)):.2f}",
            "warnings": _generate_warnings(form_data),
        }
    except Exception as e:
        raise HTTPException(status_code=400, detail=str(e))


@router.get("/previews")
def list_previews(namespace: str = "default", user: dict = Depends(get_current_user)):
    """List all SimulationPreview resources."""
    try:
        previews = k8s_client.list_simulation_previews(namespace)
        return {"previews": previews}
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))


def _generate_warnings(form_data: dict) -> list:
    warnings = []
    if form_data.get("proofMode") == "plonk-universal":
        warnings.append("PlonK proofs are Premium-tier only ($49/month)")
    if form_data.get("autonomyLevel") and form_data["autonomyLevel"] >= 4:
        warnings.append("Autonomy level 4-5 = fully autonomous. Agent decisions are advisory-only.")
    if not form_data.get("apiKey"):
        warnings.append("No API key provided. Agent will fail to start without a valid LLM API key.")
    if form_data.get("replicas", 1) > 5:
        warnings.append(f"High replica count ({form_data['replicas']}). Consider cost implications.")
    return warnings
