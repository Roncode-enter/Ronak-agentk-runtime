#!/bin/bash
set -e
export PATH="/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/usr/local/go/bin:$HOME/go/bin"

echo "Creating kind cluster 'agentk'..."
kind create cluster --name agentk

echo ""
echo "Verifying cluster..."
kubectl cluster-info --context kind-agentk
echo ""
kubectl get nodes
