#!/bin/bash
set -e
export PATH="/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/usr/local/go/bin:$HOME/go/bin"

cd /mnt/c/Users/avaka/Documents/Ronak/Ronak-agentk-runtime

echo "Running: make install"
echo "(This generates CRDs from Go structs and applies them to the cluster)"
echo ""
make install

echo ""
echo "Verifying CRDs installed:"
kubectl get crds | grep agentic-layer
