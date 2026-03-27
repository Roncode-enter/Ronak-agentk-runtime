"""Cost Intelligence router — real-time cost data from agent status."""
from fastapi import APIRouter, HTTPException, Depends
from backend.routers.auth import get_current_user
from backend.services import k8s_client

router = APIRouter()


@router.get("/{name}")
def get_cost(name: str, namespace: str = "default", user: dict = Depends(get_current_user)):
    """Get cost intelligence data for a specific agent."""
    try:
        agent = k8s_client.get_agent(name, namespace)
        status = agent.get("status", {})
        spec = agent.get("spec", {})
        budget = spec.get("costBudget", {})
        intelligence = spec.get("costIntelligence", {})

        return {
            "name": name,
            "predictedMonthlyCostUSD": status.get("predictedMonthlyCostUSD", "0.00"),
            "costAction": status.get("costAction", "none"),
            "tokensUsed": status.get("tokensUsed", 0),
            "maxMonthlyCostUSD": budget.get("maxMonthlyCostUSD", ""),
            "costPerTokenUSD": budget.get("costPerTokenUSD", ""),
            "downgradeModel": budget.get("downgradeModel", ""),
            "optimizationMode": intelligence.get("optimizationMode", ""),
            "spotInstanceFallback": intelligence.get("spotInstanceFallback", False),
            "suspendOnBudgetExhaust": intelligence.get("suspendOnBudgetExhaust", False),
        }
    except Exception as e:
        raise HTTPException(status_code=404, detail=str(e))


@router.get("")
def get_all_costs(namespace: str = "default", user: dict = Depends(get_current_user)):
    """Get cost summary for all agents."""
    try:
        agents = k8s_client.list_agents(namespace)
        costs = []
        for a in agents:
            status = a.get("status", {})
            costs.append({
                "name": a["metadata"]["name"],
                "predictedMonthlyCostUSD": status.get("predictedMonthlyCostUSD", "0.00"),
                "costAction": status.get("costAction", "none"),
                "tokensUsed": status.get("tokensUsed", 0),
            })
        return {"costs": costs}
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))
