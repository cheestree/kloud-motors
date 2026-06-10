#!/bin/bash
# reset_cloud_dbs.sh
set -e

INSTANCE_NAME="cn-db-instance"

echo "Deleting old databases on $INSTANCE_NAME..."
gcloud sql databases delete listing_db --instance=$INSTANCE_NAME --quiet || true
gcloud sql databases delete user_db --instance=$INSTANCE_NAME --quiet || true
gcloud sql databases delete chat_db --instance=$INSTANCE_NAME --quiet || true
gcloud sql databases delete seller_db --instance=$INSTANCE_NAME --quiet || true
gcloud sql databases delete auction_db --instance=$INSTANCE_NAME --quiet || true

echo "Creating new empty databases..."
gcloud sql databases create listing_db --instance=$INSTANCE_NAME
gcloud sql databases create user_db --instance=$INSTANCE_NAME
gcloud sql databases create chat_db --instance=$INSTANCE_NAME
gcloud sql databases create seller_db --instance=$INSTANCE_NAME
gcloud sql databases create auction_db --instance=$INSTANCE_NAME

echo "Databases recreated and perfectly clean!"
echo "Next step: run ./scripts/cloud/setup_cloud_db.sh to create tables and seed."