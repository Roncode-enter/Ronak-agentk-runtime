#!/bin/bash
set -e
export PATH="/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/usr/local/go/bin:$HOME/go/bin"

cd /mnt/c/Users/avaka/Documents/Ronak/Ronak-agentk-runtime

IMG="agent-runtime-operator:dev"

echo "Redeploying with updated kustomization (includes new CRDs)..."
make deploy IMG=$IMG

echo ""
echo "Restarting operator..."
kubectl rollout restart deployment agent-runtime-operator-controller-manager -n agent-runtime-operator-system
kubectl rollout status deployment agent-runtime-operator-controller-manager -n agent-runtime-operator-system --timeout=120s

echo ""
echo "Checking all CRDs (should be 14)..."
echo "Count: $(kubectl get crds | grep agentic-layer | wc -l)"
kubectl get crds | grep -E "toolsandbox|polic"

echo ""
echo "Operator pod status:"
kubectl get pods -n agent-runtime-operator-system
