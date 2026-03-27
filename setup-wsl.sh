#!/bin/bash
set -e

echo "========================================="
echo "  AgentK Dev Environment Setup for WSL"
echo "========================================="

# Step 1: Install build-essential (includes make)
echo ""
echo "[1/4] Installing build-essential (make, gcc, etc.)..."
sudo apt-get update -qq
sudo apt-get install -y -qq build-essential wget > /dev/null 2>&1
echo "  ✓ make installed: $(make --version | head -1)"

# Step 2: Install Go 1.26.0
echo ""
echo "[2/4] Installing Go 1.26.0..."
if go version 2>/dev/null | grep -q "go1.26"; then
    echo "  ✓ Go 1.26.0 already installed"
else
    sudo rm -rf /usr/local/go
    cd /tmp
    wget -q https://go.dev/dl/go1.26.0.linux-amd64.tar.gz
    sudo tar -C /usr/local -xzf go1.26.0.linux-amd64.tar.gz
    rm go1.26.0.linux-amd64.tar.gz

    # Add to PATH if not already there
    if ! grep -q '/usr/local/go/bin' ~/.bashrc; then
        echo 'export PATH=$PATH:/usr/local/go/bin:$HOME/go/bin' >> ~/.bashrc
    fi
    export PATH=$PATH:/usr/local/go/bin:$HOME/go/bin
    echo "  ✓ $(go version)"
fi

# Step 3: Install kind
echo ""
echo "[3/4] Installing kind..."
export PATH=$PATH:/usr/local/go/bin:$HOME/go/bin
if kind version 2>/dev/null; then
    echo "  ✓ kind already installed"
else
    go install sigs.k8s.io/kind@latest
    echo "  ✓ $(kind version)"
fi

# Step 4: Install operator-sdk v1.41.1
echo ""
echo "[4/4] Installing operator-sdk v1.41.1..."
if operator-sdk version 2>/dev/null | grep -q "v1.41.1"; then
    echo "  ✓ operator-sdk v1.41.1 already installed"
else
    ARCH=$(go env GOARCH)
    OS=$(go env GOOS)
    curl -sSLo /tmp/operator-sdk "https://github.com/operator-framework/operator-sdk/releases/download/v1.41.1/operator-sdk_${OS}_${ARCH}"
    chmod +x /tmp/operator-sdk
    sudo mv /tmp/operator-sdk /usr/local/bin/
    echo "  ✓ $(operator-sdk version)"
fi

echo ""
echo "========================================="
echo "  All tools installed successfully!"
echo "========================================="
echo ""
echo "Summary:"
go version
docker version --format 'Docker {{.Client.Version}}'
kubectl version --client --short 2>/dev/null || kubectl version --client | head -1
kind version
operator-sdk version | head -1
make --version | head -1
