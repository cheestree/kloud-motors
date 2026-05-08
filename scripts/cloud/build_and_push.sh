#!/bin/bash

# Configurações GCP
PROJECT_ID="cn-project-491618"
REGION="europe-central2"
REPO_NAME="vehicles"
BASE_IMAGE_URL="${REGION}-docker.pkg.dev/${PROJECT_ID}/${REPO_NAME}"

# Lista de serviços presentes na pasta code/services
SERVICES=(
  "auction"
  "auth"
  "chat"
  "gateway"
  "geographic-market-insights"
  "listing"
  "marketprice"
  "search"
  "seller"
  "user"
)

echo "🛠️ Initiating Build and Push to GCP Artifact Registry..."

for SERVICE in "${SERVICES[@]}"; do
  echo "---------------------------------------------------"
  echo "Prcessing service: ${SERVICE}..."
  
  IMAGE_TAG="${BASE_IMAGE_URL}/${SERVICE}"
  

  docker build -t ${IMAGE_TAG} -f code/services/${SERVICE}/Dockerfile code/services/
  
  echo "Pushing ${SERVICE} to GCP Artifact Registry..."
  docker push ${IMAGE_TAG}
  
  echo "✅ ${SERVICE} processed successfully!"
done

