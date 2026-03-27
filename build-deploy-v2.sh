#!/bin/bash
set -e
export PATH="/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/usr/local/go/bin:$HOME/go/bin"

cd /mnt/c/Users/avaka/Documents/Ronak/Ronak-agentk-runtime

IMG="agent-runtime-operator:dev"

echo "Step 1: Building operator Docker image..."
make docker-build IMG=$IMG

echo ""
echo "Step 2: Loading image into kind cluster 'agentk'..."
kind load docker-image $IMG --name agentk

echo ""
echo "Step 3: Deploying operator (with new ToolSandbox + Policy CRDs)..."
make deploy IMG=$IMG

echo ""
echo "Step 4: Waiting for operator pod to restart..."
kubectl rollout restart deployment agent-runtime-operator-controller-manager -n agent-runtime-operator-system
kubectl rollout status deployment agent-runtime-operator-controller-manager -n agent-runtime-operator-system --timeout=120s

echo ""
echo "Step 5: Checking CRDs..."
kubectl get crds | grep agentic-layer | wc -l
kubectl get crds | grep -E "toolsandbox|polic"
