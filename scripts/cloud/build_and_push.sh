#!/bin/bash
set -euo pipefail


PROJECT_ID="cn-project-491618"
REGION="europe-central2"
REPO_NAME="vehicles"
BASE_IMAGE_URL="${REGION}-docker.pkg.dev/${PROJECT_ID}/${REPO_NAME}"
TARGET_PLATFORM="${TARGET_PLATFORM:-linux/amd64}"


SERVICES=(
  "auction"
  "chat"
  "gateway"
  "geographic-market-insights"
  "listing"
  "marketprice"
  "redis"
  "search"
  "seller"
  "user"
)

echo "🛠️ Initiating Build and Push to GCP Artifact Registry..."
echo "📦 Target platform: ${TARGET_PLATFORM}"

if ! docker buildx version >/dev/null 2>&1; then
  echo "❌ docker buildx is required but not available."
  exit 1
fi

for SERVICE in "${SERVICES[@]}"; do
  echo "---------------------------------------------------"
  echo "Prcessing service: ${SERVICE}..."
  
  IMAGE_TAG="${BASE_IMAGE_URL}/${SERVICE}"
  
  docker buildx build \
    --platform "${TARGET_PLATFORM}" \
    -t "${IMAGE_TAG}" \
    -f "code/services/${SERVICE}/Dockerfile" \
    code/services/ \
    --push
  
  echo "✅ ${SERVICE} processed successfully!"
done
