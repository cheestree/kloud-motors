#!/usr/bin/env bash

PROJECT_ID="cn-project-491618"
SA_NAME="github-actions-deploy"
SA_EMAIL="${SA_NAME}@${PROJECT_ID}.iam.gserviceaccount.com"

gcloud iam service-accounts create $SA_NAME \
  --project=$PROJECT_ID \
  --display-name="GitHub Actions Deploy Service Account"

gcloud projects add-iam-policy-binding $PROJECT_ID \
  --member=serviceAccount:$SA_EMAIL \
  --role=roles/artifactregistry.writer

gcloud projects add-iam-policy-binding $PROJECT_ID \
  --member=serviceAccount:$SA_EMAIL \
  --role=roles/container.developer

gcloud projects add-iam-policy-binding $PROJECT_ID \
  --member=serviceAccount:$SA_EMAIL \
  --role=roles/cloudsql.client

gcloud projects add-iam-policy-binding $PROJECT_ID \
  --member=serviceAccount:$SA_EMAIL \
  --role=roles/cloudsql.instanceUser

gcloud projects add-iam-policy-binding $PROJECT_ID \
  --member=serviceAccount:$SA_EMAIL \
  --role=roles/pubsub.admin

gcloud projects add-iam-policy-binding $PROJECT_ID \
  --member=serviceAccount:$SA_EMAIL \
  --role=roles/iam.serviceAccountUser

# Create a minimal custom IAM role to allow patching ValidatingWebhookConfigurations
# (permission required so the deploy SA can apply nginx ingress webhook patches)
gcloud iam roles create webhookPatchRole --project="$PROJECT_ID" \
  --title="Webhook Patch Role" \
  --permissions="container.validatingWebhookConfigurations.update" \
  --stage="GA" || echo "webhookPatchRole already exists or creation failed, continuing"

gcloud projects add-iam-policy-binding $PROJECT_ID \
  --member=serviceAccount:$SA_EMAIL \
  --role="projects/${PROJECT_ID}/roles/webhookPatchRole" || echo "binding webhookPatchRole failed or already exists"

# Create and download a key for the service account
gcloud iam service-accounts keys create ./gcp-sa-key.json \
  --iam-account=$SA_EMAIL

cat ./gcp-sa-key.json