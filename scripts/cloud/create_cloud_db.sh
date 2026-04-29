#!/bin/bash

INSTANCE_NAME="cn-db-instance"

echo "Creating Cloud SQL instance $INSTANCE_NAME..."

gcloud sql instances create $INSTANCE_NAME \
    --database-version=POSTGRES_15 \
    --tier=db-f1-micro \
    --region=europe-central2 \
    --root-password=uma_password_forte_aqui

echo "Instance created! Now creating databases..."

gcloud sql databases create listing_db --instance=cn-db-instance
gcloud sql databases create auth_db --instance=cn-db-instance
gcloud sql databases create user_db --instance=cn-db-instance
gcloud sql databases create seller_db --instance=cn-db-instance
gcloud sql databases create auction_db --instance=cn-db-instance
gcloud sql databases create chat_db --instance=cn-db-instance

echo "All done! The Cloud SQL instance and databases have been created."
echo "Next step: run ./scripts/cloud/setup_cloud_db.sh to create tables and seed."