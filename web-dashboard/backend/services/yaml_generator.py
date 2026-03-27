"""Generates Agent YAML from dashboard form data."""
import yaml


def generate_agent_yaml(form_data: dict) -> dict:
    """Convert dashboard form fields into a valid Agent CR dict."""
    agent = {
        "apiVersion": "runtime.agentic-layer.ai/v1alpha1",
        "kind": "Agent",
        "metadata": {
            "name": form_data["name"],
            "namespace": form_data.get("namespace", "default"),
        },
        "spec": {
            "framework": form_data.get("framework", "google-adk"),
            "description": form_data.get("description", ""),
            "instruction": form_data.get("instruction", ""),
            "protocols": [{"type": "A2A"}],
            "replicas": form_data.get("replicas", 1),
        },
    }

    spec = agent["spec"]

    if form_data.get("image"):
        spec["image"] = form_data["image"]

    if form_data.get("model"):
        spec["model"] = form_data["model"]

    # Environment variables
    env = []
    if form_data.get("apiKey"):
        env.append({"name": "GEMINI_API_KEY", "value": form_data["apiKey"]})
    if env:
        spec["env"] = env

    # Cost Budget
    if form_data.get("maxMonthlyCost"):
        spec["costBudget"] = {
            "maxMonthlyCostUSD": str(form_data["maxMonthlyCost"]),
            "costPerTokenUSD": form_data.get("costPerToken", "0.00001"),
        }
        if form_data.get("downgradeModel"):
            spec["costBudget"]["downgradeModel"] = form_data["downgradeModel"]

    # Cost Intelligence
    if form_data.get("optimizationMode"):
        spec["costIntelligence"] = {
            "optimizationMode": form_data["optimizationMode"],
            "spotInstanceFallback": form_data.get("spotFallback", False),
            "suspendOnBudgetExhaust": form_data.get("suspendOnExhaust", False),
        }

    # Verifiable
    if form_data.get("verifiableEnabled"):
        spec["verifiable"] = {
            "enabled": True,
            "proofMode": form_data.get("proofMode", "snark-groth16"),
        }

    # Governance
    if form_data.get("autonomyLevel"):
        spec["governance"] = {
            "autonomyLevel": form_data["autonomyLevel"],
            "requirePolicyCompliance": form_data.get("requireCompliance", True),
        }
        if form_data.get("humanWebhook"):
            spec["governance"]["humanApprovalWebhook"] = form_data["humanWebhook"]

    # Lifecycle
    if form_data.get("strategy"):
        spec["lifecycle"] = {
            "strategy": form_data["strategy"],
            "selfHealing": form_data.get("selfHealing", True),
        }
        if form_data.get("promptVersion"):
            spec["lifecycle"]["promptVersion"] = form_data["promptVersion"]

    return agent
