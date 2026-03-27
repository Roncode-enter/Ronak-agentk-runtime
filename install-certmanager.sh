#!/bin/bash
set -e
export PATH="/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/usr/local/go/bin:$HOME/go/bin"

echo "Installing cert-manager v1.19.1..."
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.19.1/cert-manager.yaml

echo ""
echo "Waiting for cert-manager pods to be ready (this may take 1-2 minutes)..."
kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=cert-manager -n cert-manager --timeout=120s
kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=webhook -n cert-manager --timeout=120s
kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=cainjector -n cert-manager --timeout=120s

echo ""
echo "cert-manager pods:"
kubectl get pods -n cert-manager
