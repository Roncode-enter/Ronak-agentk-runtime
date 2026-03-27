#!/bin/bash
set -e
export PATH="/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/usr/local/go/bin:$HOME/go/bin"

cd /mnt/c/Users/avaka/Documents/Ronak/Ronak-agentk-runtime

echo "============================================"
echo "  Applying example resources..."
echo "============================================"

echo ""
echo "--- Applying ToolSandbox ---"
kubectl apply -f examples/toolsandbox-basic.yaml

echo ""
echo "--- Applying Policies ---"
kubectl apply -f examples/policy-pii.yaml
kubectl apply -f examples/policy-spend.yaml
kubectl apply -f examples/policy-network.yaml

echo ""
echo "--- Applying Agent with Sandbox + Policy ---"
kubectl apply -f examples/agent-with-sandbox.yaml

echo ""
echo "Waiting 10 seconds for reconciliation..."
sleep 10

echo ""
echo "============================================"
echo "  Verification Results"
echo "============================================"

echo ""
echo "--- ToolSandboxes ---"
kubectl get toolsandbox
echo ""
kubectl describe toolsandbox code-sandbox | tail -15

echo ""
echo "--- Policies ---"
kubectl get policy
echo ""

echo ""
echo "--- Agent with Sandbox ---"
kubectl get agent secure-coding-assistant

echo ""
echo "--- Operator Logs (last 30 lines) ---"
kubectl logs -n agent-runtime-operator-system -l control-plane=controller-manager --tail=30

echo ""
echo "============================================"
echo "  All done!"
echo "============================================"
