#!/bin/bash
export PATH="/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/usr/local/go/bin:$HOME/go/bin"

echo "Forwarding localhost:8000 -> hello-agentk pod port 8000"
echo ""
echo "Open in your browser:  http://localhost:8000/.well-known/agent-card.json"
echo ""
echo "Press Ctrl+C to stop."
echo ""
kubectl port-forward svc/hello-agentk 8000:8000
