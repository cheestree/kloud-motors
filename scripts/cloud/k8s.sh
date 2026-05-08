#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
K8S_DIR="$ROOT_DIR/deploy/k8s"
KUSTOMIZE_DIR="$K8S_DIR"
INGRESS_CONTROLLER_MANIFEST="$K8S_DIR/nginx-controller.yaml"
GATEWAY_MANIFEST="$K8S_DIR/gateway/gateway.yaml"
INGRESS_MANIFEST="$K8S_DIR/ingress.yaml"
NAMESPACE_FILE="$K8S_DIR/common/namespace.yaml"

ACTION="up"
WAIT_FOR_ROLLOUT=true
WITH_INGRESS=false

usage() {
  cat <<EOF
Usage: ./k8s.sh [up|down|status|restart] [--with-ingress] [--no-wait]

Commands:
  up              Apply Kubernetes manifests (default)
  down            Delete Kubernetes manifests
  status          Show current pod and service status
  restart         Restart all deployments (rollout restart)

Flags:
  --with-ingress  Also apply/delete deploy/k8s/ingress.yaml
  --no-wait       Do not wait for deployments to become available (up only)
  -h, --help      Show this help
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    up|down|status|restart)
      ACTION="$1"
      ;;
    --with-ingress)
      WITH_INGRESS=true
      ;;
    --no-wait)
      WAIT_FOR_ROLLOUT=false
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "Unknown argument: $1"
      usage
      exit 1
      ;;
  esac
  shift
done

if ! command -v kubectl >/dev/null 2>&1; then
  echo "kubectl not found. Install kubectl and configure cluster access first."
  exit 1
fi

if [[ ! -f "$NAMESPACE_FILE" ]]; then
  echo "Namespace file not found: $NAMESPACE_FILE"
  exit 1
fi

NAMESPACE="$(awk '/^  name:/ {print $2; exit}' "$NAMESPACE_FILE")"
if [[ -z "$NAMESPACE" ]]; then
  echo "Could not read namespace from: $NAMESPACE_FILE"
  exit 1
fi

KUBECONFIG_FILE=""
KUBE_ARGS=()

# Prefer KUBECONFIG env var (set by GitHub Actions get-gke-credentials)
if [[ -n "${KUBECONFIG:-}" ]] && [[ -f "$KUBECONFIG" ]]; then
  KUBECONFIG_FILE="$KUBECONFIG"
  echo "Using kubeconfig from KUBECONFIG env var: $KUBECONFIG_FILE"
# Fall back to project kubeconfig for local runs
elif [[ -f "$K8S_DIR/kubeconfig" ]]; then
  KUBECONFIG_FILE="$K8S_DIR/kubeconfig"
  KUBE_ARGS+=(--kubeconfig="$KUBECONFIG_FILE")
  echo "Using project kubeconfig: $KUBECONFIG_FILE"
else
  echo "Using default kubeconfig (~/.kube/config)"
fi

if [[ -n "$KUBECONFIG_FILE" ]]; then
  KUBE_ARGS+=(--kubeconfig="$KUBECONFIG_FILE")
fi

k() {
  kubectl "${KUBE_ARGS[@]}" "$@"
}

if ! k cluster-info >/dev/null 2>&1; then
  echo "Cannot connect to the Kubernetes cluster. Check your kubeconfig/context."
  exit 1
fi

apply_up() {
  if [[ "$WITH_INGRESS" == true ]]; then
    echo "Applying ingress controller manifest..."
    k apply -f "$INGRESS_CONTROLLER_MANIFEST"

    echo "Waiting for ingress controller rollout..."
    k -n ingress-nginx rollout status deployment/ingress-nginx-controller --timeout=600s
  fi

  echo "Applying application manifests with kustomize..."
  k apply -k "$KUSTOMIZE_DIR"

  if [[ "$WITH_INGRESS" == true ]]; then
    echo "Applying ingress manifest..."
    k -n "$NAMESPACE" apply -f "$INGRESS_MANIFEST"
  fi

  echo "Restarting deployments so pods pull the latest image..."
  while IFS= read -r deployment; do
    [[ -z "$deployment" ]] && continue
    k -n "$NAMESPACE" rollout restart "$deployment"
    if [[ "$WAIT_FOR_ROLLOUT" == true ]]; then
      k -n "$NAMESPACE" rollout status "$deployment" --timeout=300s
    fi
  done < <(k -n "$NAMESPACE" get deployments -o name)

  if [[ "$WAIT_FOR_ROLLOUT" == true ]]; then
    echo "Waiting for deployments in namespace $NAMESPACE..."
    k -n "$NAMESPACE" wait --for=condition=available deployment --all --timeout=300s
  fi

  echo "Kubernetes deployment is up."
}

apply_down() {
  echo "Deleting manifests..."

  if [[ "$WITH_INGRESS" == true ]]; then
    k -n "$NAMESPACE" delete -f "$INGRESS_MANIFEST" --ignore-not-found
    k delete -f "$INGRESS_CONTROLLER_MANIFEST" --ignore-not-found
  fi

  k delete -k "$KUSTOMIZE_DIR" --ignore-not-found

  echo "Kubernetes deployment removed."
}

show_status() {
  echo "Namespace: $NAMESPACE"
  k get namespace "$NAMESPACE" >/dev/null 2>&1 || {
    echo "Namespace $NAMESPACE does not exist."
    exit 0
  }

  echo
  echo "Pods:"
  k -n "$NAMESPACE" get pods -o wide

  echo
  echo "Services:"
  k -n "$NAMESPACE" get services

  echo
  echo "Deployments:"
  k -n "$NAMESPACE" get deployments
}

apply_restart() {
  echo "Restarting all deployments in namespace $NAMESPACE..."
  # Restart each deployment individually for better compatibility
  k -n "$NAMESPACE" get deployments -o name | while read -r deploy; do
    k -n "$NAMESPACE" rollout restart "$deploy"
  done
  
  if [[ "$WAIT_FOR_ROLLOUT" == true ]]; then
    echo "Waiting for rollout to complete..."
    k -n "$NAMESPACE" get deployments -o name | while read -r deploy; do
      k -n "$NAMESPACE" rollout status "$deploy"
    done
  fi

  echo "All services restarted."
}

case "$ACTION" in
  up)
    apply_up
    ;;
  down)
    apply_down
    ;;
  status)
    show_status
    ;;
  restart)
    apply_restart
    ;;
esac
