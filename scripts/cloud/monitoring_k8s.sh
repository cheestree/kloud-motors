#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
K8S_DIR="$ROOT_DIR/deploy/k8s"
KUSTOMIZE_DIR="$K8S_DIR/monitoring"
INGRESS_FILE="$K8S_DIR/ingress_monitoring.yaml"
NAMESPACE_FILE="$KUSTOMIZE_DIR/namespace.yaml"

ACTION="up"
WAIT_FOR_ROLLOUT=true
WITH_INGRESS=false

usage() {
  cat <<EOF
Usage: ./monitoring_k8s.sh [up|down|status|restart] [--no-wait]

Commands:
  up              Apply monitoring manifests (default)
  down            Delete monitoring manifests
  status          Show monitoring pod and service status
  restart         Restart monitoring deployments/daemonsets

Flags:
  --no-wait       Do not wait for deployments to become available (up/restart only)
  --with-ingress Apply monitoring ingress manifest
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

KUBECONFIG_FILE="$K8S_DIR/kubeconfig"
KUBE_ARGS=()
if [[ -f "$KUBECONFIG_FILE" ]]; then
  KUBE_ARGS+=(--kubeconfig="$KUBECONFIG_FILE")
  echo "Using project kubeconfig: $KUBECONFIG_FILE"
else
  echo "Using default kubeconfig (~/.kube/config)"
fi

k() {
  kubectl "${KUBE_ARGS[@]}" "$@"
}

if ! k cluster-info >/dev/null 2>&1; then
  echo "Cannot connect to the Kubernetes cluster. Check your kubeconfig/context."
  exit 1
fi

apply_up() {
  echo "Applying monitoring manifests with kustomize..."
  k apply -k "$KUSTOMIZE_DIR"

  if [[ "$WITH_INGRESS" == true ]]; then
    if [[ -f "$INGRESS_FILE" ]]; then
      echo "Applying monitoring ingress: $INGRESS_FILE"
      k apply -f "$INGRESS_FILE"
    else
      echo "Monitoring ingress file not found: $INGRESS_FILE"
    fi
  fi

  if [[ "$WAIT_FOR_ROLLOUT" == true ]]; then
    echo "Waiting for deployments in namespace $NAMESPACE..."
    k -n "$NAMESPACE" wait --for=condition=available deployment --all --timeout=300s
  fi

  echo "Monitoring stack is up."
}

apply_down() {
  echo "Deleting monitoring manifests..."
  k delete -k "$KUSTOMIZE_DIR" --ignore-not-found

  if [[ "$WITH_INGRESS" == true ]]; then
    INGRESS_FILE="$K8S_DIR/ingress_monitoring.yaml"
    if [[ -f "$INGRESS_FILE" ]]; then
      echo "Deleting monitoring ingress: $INGRESS_FILE"
      k delete -f "$INGRESS_FILE" --ignore-not-found
    fi
  fi

  echo "Monitoring stack removed."
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

  echo
  echo "DaemonSets:"
  k -n "$NAMESPACE" get daemonsets
}

apply_restart() {
  echo "Restarting monitoring deployments and daemonsets in namespace $NAMESPACE..."
  k -n "$NAMESPACE" get deployments -o name | while read -r deploy; do
    [[ -z "$deploy" ]] && continue
    k -n "$NAMESPACE" rollout restart "$deploy"
  done

  k -n "$NAMESPACE" get daemonsets -o name | while read -r ds; do
    [[ -z "$ds" ]] && continue
    k -n "$NAMESPACE" rollout restart "$ds"
  done

  if [[ "$WAIT_FOR_ROLLOUT" == true ]]; then
    echo "Waiting for rollout to complete..."
    k -n "$NAMESPACE" get deployments -o name | while read -r deploy; do
      [[ -z "$deploy" ]] && continue
      k -n "$NAMESPACE" rollout status "$deploy"
    done
  fi

  echo "Monitoring services restarted."
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
