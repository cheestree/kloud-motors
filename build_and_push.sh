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

echo "🛠️ A iniciar Build e Push para o GCP Artifact Registry..."

for SERVICE in "${SERVICES[@]}"; do
  echo "---------------------------------------------------"
  echo "🚀 Processando serviço: ${SERVICE}..."
  
  IMAGE_TAG="${BASE_IMAGE_URL}/${SERVICE}"
  
  # Fazer o Build da imagem (contexto na raiz do code/services para ler shared/ se precisar, ou na pasta do serviço)
  # Como os Dockerfiles estão dentro da pasta de cada serviço:
  echo "📦 A fazer build de ${IMAGE_TAG}..."
  docker build -t ${IMAGE_TAG} -f code/services/${SERVICE}/Dockerfile code/services/
  
  # Empurrar para a Google Cloud
  echo "☁️ A fazer push para GCP..."
  docker push ${IMAGE_TAG}
  
  echo "✅ ${SERVICE} concluído com sucesso!"
done

echo "🎉 Todas as imagens foram processadas!"
