#!/bin/bash
set -e
export PATH="/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/usr/local/go/bin:$HOME/go/bin"

echo "Deploying 'Hello from AgentK' agent..."
kubectl apply -f /mnt/c/Users/avaka/Documents/Ronak/Ronak-agentk-runtime/hello-agent.yaml

echo ""
echo "Waiting for agent resources to be created..."
sleep 5

echo ""
echo "=== Agent Resource ==="
kubectl get agent hello-agentk

echo ""
echo "=== Deployment ==="
kubectl get deployment hello-agentk

echo ""
echo "=== Service ==="
kubectl get service hello-agentk

echo ""
echo "=== Agent Details ==="
kubectl describe agent hello-agentk
