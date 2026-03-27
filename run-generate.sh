#!/bin/bash
set -e
export PATH="/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/usr/local/go/bin:$HOME/go/bin"

cd /mnt/c/Users/avaka/Documents/Ronak/Ronak-agentk-runtime

echo "Step 1: make generate (creates DeepCopy methods for new types)..."
make generate

echo ""
echo "Step 2: make manifests (generates CRD YAML, RBAC, webhook configs)..."
make manifests

echo ""
echo "Step 3: Verifying generated CRD files..."
ls -la config/crd/bases/ | grep -E "toolsandbox|polic"

echo ""
echo "Done! New CRDs generated."
