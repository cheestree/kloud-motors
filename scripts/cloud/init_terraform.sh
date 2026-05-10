#!/bin/bash
# scripts/cloud/init_terraform.sh
set -e

echo "Starting Terraform Bootstrap..."

# Unset the credentials variable if it exists to force Application Default Credentials
unset GOOGLE_APPLICATION_CREDENTIALS

# Obter o Project ID atual configurado no gcloud
PROJECT_ID=$(gcloud config get-value project)

if [ -z "$PROJECT_ID" ]; then
    echo "Error: Could not determine GCP Project ID. Please run 'gcloud config set project <your_project_id>' first."
    exit 1
fi

echo "Using GCP Project: $PROJECT_ID"

# Nome do bucket padronizado baseado no Project ID
BUCKET_NAME="cn-terraform-state-$PROJECT_ID"
LOCATION="europe-central2"

# Verificar se o bucket já existe
if gcloud storage buckets describe "gs://$BUCKET_NAME" >/dev/null 2>&1; then
    echo "Terraform state bucket 'gs://$BUCKET_NAME' already exists."
else
    echo "Terraform state bucket 'gs://$BUCKET_NAME' does not exist. Creating it now..."
    gcloud storage buckets create "gs://$BUCKET_NAME" --project="$PROJECT_ID" --location="$LOCATION" --uniform-bucket-level-access
    
    # Enable versioning (Recommended for Terraform state)
    gcloud storage buckets update "gs://$BUCKET_NAME" --versioning
    
    echo "Bucket created successfully!"
fi

echo "Initializing Terraform..."
cd "$(dirname "$0")/../../terraform"

# Inicializar o terraform injetando o bucket dinamicamente
terraform init -backend-config="bucket=$BUCKET_NAME"

echo "Terraform initialization complete!"
