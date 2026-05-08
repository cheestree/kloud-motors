#!/usr/bin/env bash
set -euo pipefail

# setup-all-env.sh
# Read .env.secrets and .env.variables, set variables in GitHub Actions as appropriate.
# Usage:
#   ./scripts/github/setup-all-env.sh [--repo owner/repo] [--dry-run]

REPO="cheestree/comp-nuvem-2026"
DRY_RUN=false

while [[ $# -gt 0 ]]; do
  case "$1" in
    --repo) REPO="$2"; shift 2;;
    --dry-run|-n) DRY_RUN=true; shift ;;
    -h|--help) echo "Usage: $0 [--repo owner/repo] [--dry-run]"; exit 0;;
    *) echo "Unknown arg: $1"; exit 1;;
  esac
done

if ! command -v gh >/dev/null 2>&1; then
  echo "gh CLI not found. Install and run 'gh auth login' first." >&2
  exit 2
fi

if [ -z "$REPO" ]; then
  if REPO=$(gh repo view --json nameWithOwner -q . 2>/dev/null); then
    echo "Detected repo: $REPO"
  else
    echo "Repository not provided and gh couldn't detect it. Use --repo owner/repo" >&2
    exit 3
  fi
fi

count=0

set_item() {
  local kind="$1"
  local key="$2"
  local value="$3"

  if [ "$DRY_RUN" = true ]; then
    echo "  would set $kind: $key"
    return 0
  fi

  if [ "$kind" = "secret" ]; then
    gh secret set "$key" --body "$value" --repo "$REPO"
  else
    gh variable set "$key" --body "$value" --repo "$REPO"
  fi

  echo "  set $kind: $key"
}

# Process both files
for env_file in .env.variables .env.secrets; do
  if [ ! -f "$env_file" ]; then
    echo "Warning: $env_file not found, skipping."
    continue
  fi

  echo "Processing $env_file..."
  kind="variable"
  [ "$env_file" = ".env.secrets" ] && kind="secret"

  while IFS= read -r raw_line || [ -n "$raw_line" ]; do
    # normalize and trim
    line="${raw_line%$'\r'}"
    line="$(echo "$line" | sed -E 's/^[[:space:]]+//;s/[[:space:]]+$//')"
    [[ -z "$line" ]] && continue
    [[ "$line" =~ ^# ]] && continue

    # parse key=value (with optional export prefix)
    if [[ "$line" =~ ^export[[:space:]]+([A-Za-z_][A-Za-z0-9_]*)=(.*)$ ]]; then
      key="${BASH_REMATCH[1]}"
      value="${BASH_REMATCH[2]}"
    elif [[ "$line" =~ ^([A-Za-z_][A-Za-z0-9_]*)=(.*)$ ]]; then
      key="${BASH_REMATCH[1]}"
      value="${BASH_REMATCH[2]}"
    else
      continue
    fi

    # remove surrounding quotes
    if [[ "$value" =~ ^\".*\"$ || "$value" =~ ^\'.*\'$ ]]; then
      value="${value:1:${#value}-2}"
    else
      # remove inline comment for unquoted values
      value="$(echo "$value" | sed -E 's/[[:space:]]*#.*$//')"
      value="$(echo -n "$value" | sed -E 's/[[:space:]]+$//')"
    fi

    set_item "$kind" "$key" "$value"
    count=$((count+1))
  done < "$env_file"
done

echo "Done. Total items set: $count"
