#!/bin/bash
set -e
export PATH="/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/usr/local/go/bin:$HOME/go/bin"

cd /mnt/c/Users/avaka/Documents/Ronak/Ronak-agentk-runtime

IMG="agent-runtime-operator:dev"

echo "Step 1: Building operator Docker image ($IMG)..."
make docker-build IMG=$IMG

echo ""
echo "Step 2: Loading image into kind cluster 'agentk'..."
kind load docker-image $IMG --name agentk

echo ""
echo "Step 3: Deploying operator to cluster..."
make deploy IMG=$IMG

echo ""
echo "Step 4: Waiting for operator pod to be ready..."
kubectl wait --for=condition=ready pod -l control-plane=controller-manager -n agent-runtime-operator-system --timeout=120s

echo ""
echo "Operator pods:"
kubectl get pods -n agent-runtime-operator-system
