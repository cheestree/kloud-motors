#!/bin/bash
# init_cluster.sh
# This script is used when we are creating a new cluster
set -e

REPO_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"

echo "Installing the Ingress Controller (Network Engine) in the cluster..."
kubectl apply -f "$REPO_ROOT/deploy/k8s/nginx-controller.yaml"

echo "Waiting for Google Cloud to assign a public IP (may take 2 min)..."
kubectl wait --namespace ingress-nginx \
  --for=condition=ready pod \
  --selector=app.kubernetes.io/component=controller \
  --timeout=120s

echo "The network engine is installed! You can now run ./scripts/cloud/k8s.sh up"