# Variáveis
PROJECT=cn-project-491618
SA=vehicles-svc@cn-project-491618.iam.gserviceaccount.com
KSA=vehicles-prod/vehicles-workload-sa  # namespace/k8s-sa-name

# 1. Pub/Sub (chat - publish + subscribe)
gcloud projects add-iam-policy-binding $PROJECT \
  --member="serviceAccount:$SA" \
  --role="roles/pubsub.publisher"

gcloud projects add-iam-policy-binding $PROJECT \
  --member="serviceAccount:$SA" \
  --role="roles/pubsub.subscriber"

# 2. Firestore (chat - leitura/escrita de mensagens)
gcloud projects add-iam-policy-binding $PROJECT \
  --member="serviceAccount:$SA" \
  --role="roles/datastore.user"

# 3. Cloud SQL (todos os serviços - via proxy sidecar)
gcloud projects add-iam-policy-binding $PROJECT \
  --member="serviceAccount:$SA" \
  --role="roles/cloudsql.client"

# 4. Workload Identity binding (GKE SA → GCP SA)
gcloud iam service-accounts add-iam-policy-binding $SA \
  --project=$PROJECT \
  --role="roles/iam.workloadIdentityUser" \
  --member="serviceAccount:$PROJECT.svc.id.goog[vehicles-prod/vehicles-workload-sa]"

# 5. Artifact Registry (para o node pool fazer pull das imagens)
gcloud projects add-iam-policy-binding $PROJECT \
  --member="serviceAccount:$SA" \
  --role="roles/artifactregistry.reader"

# 6. Cloud Trace (para o node pool enviar traces)
gcloud projects add-iam-policy-binding $PROJECT \
  --member="serviceAccount:$SA" \
  --role="roles/cloudtrace.agent"
