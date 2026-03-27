#!/bin/bash
export PATH="/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/usr/local/go/bin:$HOME/go/bin"

echo "============================================"
echo "  AgentK Runtime - Full Verification"
echo "============================================"

echo ""
echo "--- Tools ---"
echo "Go:           $(go version 2>/dev/null | awk '{print $3}')"
echo "Docker:       $(docker version --format '{{.Client.Version}}' 2>/dev/null)"
echo "kubectl:      $(kubectl version --client 2>/dev/null | head -1)"
echo "kind:         $(kind version 2>/dev/null)"
echo "operator-sdk: $(operator-sdk version 2>/dev/null | cut -d'"' -f2)"
echo "make:         $(make --version 2>/dev/null | head -1)"

echo ""
echo "--- Kind Cluster ---"
echo "Clusters: $(kind get clusters 2>/dev/null)"
kubectl get nodes

echo ""
echo "--- cert-manager ---"
kubectl get pods -n cert-manager

echo ""
echo "--- Operator ---"
kubectl get pods -n agent-runtime-operator-system

echo ""
echo "--- CRDs (12 expected) ---"
kubectl get crds | grep -c agentic-layer
kubectl get crds | grep agentic-layer

echo ""
echo "--- Hello AgentK Agent ---"
kubectl get agent hello-agentk
kubectl get deployment hello-agentk
kubectl get pods -l app=hello-agentk

echo ""
echo "--- Agent Service URL ---"
kubectl get agent hello-agentk -o jsonpath='{.status.url}'
echo ""

echo ""
echo "============================================"
echo "  All done!"
echo "============================================"
