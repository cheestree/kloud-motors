#!/usr/bin/env bash

# remove-all-env.sh
# Remove all secrets and variables from the GitHub repository.
# Usage:
#   ./scripts/github/remove-all-env.sh 

echo "This will delete ALL secrets and variables from the GitHub repository. Are you sure? (y/N)"
read -r confirm
if [[ "$confirm" != "y" ]]; then
  echo "Aborting."
  exit 0
fi

echo "Deleting all secrets..."
gh secret list --repo cheestree/comp-nuvem-2026 | awk 'NR>1 {print $1}' | xargs -r -n1 gh secret delete --repo cheestree/comp-nuvem-2026

echo "Deleting all variables..."
gh variable list --repo cheestree/comp-nuvem-2026 | awk 'NR>1 {print $1}' | xargs -r -n1 gh variable delete --repo cheestree/comp-nuvem-2026

echo "All secrets and variables deleted."