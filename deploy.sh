#!/bin/bash
set -e

# Check for kubectl
if ! command -v kubectl &> /dev/null; then
  echo "kubectl not found. Please install kubectl and configure your cluster."
  exit 1
fi


# Use project-local kubeconfig if present
KUBECONFIG_FILE="deploy/k8s/kubeconfig"
if [ -f "$KUBECONFIG_FILE" ]; then
  export KUBECONFIG="$KUBECONFIG_FILE"
  echo "Using project-local kubeconfig: $KUBECONFIG_FILE"
  KUBECONFIG_FLAG="--kubeconfig=$KUBECONFIG_FILE"
else
  echo "Using default kubeconfig (~/.kube/config)"
  KUBECONFIG_FLAG=""
fi


# Check cluster connectivity
if ! kubectl $KUBECONFIG_FLAG cluster-info &> /dev/null; then
  echo "Cannot connect to Kubernetes cluster. Please check your kubeconfig."
  exit 1
fi


# Apply namespace and secrets
kubectl $KUBECONFIG_FLAG apply -f deploy/k8s/common/namespace.yaml
kubectl $KUBECONFIG_FLAG apply -f deploy/k8s/common/secrets.yaml

# Apply all manifests using kustomize
kubectl $KUBECONFIG_FLAG apply -k deploy/k8s/

echo "Deployment complete!"
