#!/bin/bash
set -e
export PATH="/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/usr/local/go/bin:$HOME/go/bin"

cd /mnt/c/Users/avaka/Documents/Ronak/Ronak-agentk-runtime

IMG="agent-runtime-operator:dev"

echo "============================================"
echo "  Phase 2.3: Swarm CRD + Edge Profile"
echo "============================================"

echo ""
echo "Step 1: Building operator Docker image..."
make docker-build IMG=$IMG

echo ""
echo "Step 2: Loading image into kind cluster 'agentk'..."
kind load docker-image $IMG --name agentk

echo ""
echo "Step 3: Deploying operator (with Swarm CRD)..."
make deploy IMG=$IMG

echo ""
echo "Step 4: Restarting operator pod..."
kubectl rollout restart deployment agent-runtime-operator-controller-manager -n agent-runtime-operator-system
kubectl rollout status deployment agent-runtime-operator-controller-manager -n agent-runtime-operator-system --timeout=120s

echo ""
echo "Step 5: Checking CRDs (should be 16 — 15 previous + swarms)..."
echo "Count: $(kubectl get crds | grep agentic-layer | wc -l)"
echo ""
echo "Swarm CRD:"
kubectl get crd swarms.runtime.agentic-layer.ai 2>&1 || echo "WARNING: Swarm CRD not found!"

echo ""
echo "Step 6: Applying example resources..."
kubectl apply -f examples/simple-swarm.yaml 2>&1
kubectl apply -f examples/edge-agent.yaml 2>&1

echo ""
echo "Waiting 10 seconds for reconciliation..."
sleep 10

echo ""
echo "Step 7: Checking results..."
echo "=== All Agents ==="
kubectl get agent
echo ""
echo "=== All Swarms ==="
kubectl get swarm
echo ""
echo "=== Deployments ==="
kubectl get deploy
echo ""
echo "=== Services ==="
kubectl get svc
echo ""
echo "=== ConfigMaps (coordinator) ==="
kubectl get configmap | grep coordinator || echo "(none yet)"

echo ""
echo "Step 8: Swarm details..."
kubectl describe swarm simple-swarm 2>&1 | head -40

echo ""
echo "Step 9: Operator logs (last 20 lines)..."
kubectl logs -l control-plane=controller-manager -n agent-runtime-operator-system --tail=20

echo ""
echo "============================================"
echo "  Phase 2.3 DEPLOYMENT COMPLETE!"
echo "============================================"
