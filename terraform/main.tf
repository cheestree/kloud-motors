resource "google_sql_database_instance" "db_instance" {
  name             = "cn-db-instance"
  database_version = "POSTGRES_15"
  region           = var.region
  root_password    = var.db_password
  settings { tier = "db-f1-micro" }
}

resource "google_sql_database" "databases" {
  for_each = toset(var.databases)
  name     = each.value
  instance = google_sql_database_instance.db_instance.name
}

resource "google_storage_bucket" "backup_bucket" {
  name                        = var.backup_bucket_name
  location                    = var.region
  uniform_bucket_level_access = true

  versioning {
    enabled = true
  }

  lifecycle_rule {
    condition {
      age = 30
    }
    action {
      type = "SetStorageClass"
      storage_class = "COLDLINE"
    }
  }
}

resource "google_storage_bucket_iam_member" "cloud_sql_backup_writer" {
  bucket = google_storage_bucket.backup_bucket.name
  role   = "roles/storage.objectAdmin"
  member = "serviceAccount:${google_sql_database_instance.db_instance.service_account_email_address}"
}

# Artifact Registry para as imagens Docker
resource "google_artifact_registry_repository" "vehicles_repo" {
  location      = var.region
  repository_id = var.artifact_registry_repo
  description   = "Docker repository for microservices"
  format        = "DOCKER"
}

# Pub/Sub Topics
resource "google_pubsub_topic" "topics" {
  for_each = toset(var.pubsub_topics)
  name     = each.value
}


resource "google_service_account" "vehicles_sa" {
  account_id   = "vehicles-svc"
  display_name = "Service Account para Microserviços"
}

resource "google_project_iam_member" "roles" {
  for_each = toset([
    "roles/pubsub.publisher", "roles/pubsub.subscriber",
    "roles/datastore.user", "roles/cloudsql.client",
    "roles/artifactregistry.reader", "roles/storage.objectAdmin",
    "roles/cloudsql.admin"
  ])
  project = var.project_id
  role    = each.value
  member  = "serviceAccount:${google_service_account.vehicles_sa.email}"
}

resource "google_compute_network" "vpc_network" {
  name                    = "cn-vpc-network"
  auto_create_subnetworks = false
}

resource "google_compute_subnetwork" "subnet" {
  name          = "cn-subnetwork"
  ip_cidr_range = "10.0.0.0/16"
  region        = var.region
  network       = google_compute_network.vpc_network.id
}

resource "google_container_cluster" "primary" {
  name       = "cn-cluster"
  location   = var.region
  network    = google_compute_network.vpc_network.id
  subnetwork = google_compute_subnetwork.subnet.id

  remove_default_node_pool = true
  initial_node_count       = 1

  node_config {
    disk_size_gb = 15
  }

  # Habilitar Workload Identity
  workload_identity_config {
    workload_pool = "${var.project_id}.svc.id.goog"
  }
}

resource "google_container_node_pool" "primary_nodes" {
  name               = "primary-node-pool"
  cluster            = google_container_cluster.primary.id
  initial_node_count = var.node_pool_min_node_count

  autoscaling {
    min_node_count = var.node_pool_min_node_count
    max_node_count = var.node_pool_max_node_count
  }

  node_config {
    machine_type = "e2-standard-2"
    disk_size_gb = 30
    oauth_scopes = [
      "https://www.googleapis.com/auth/cloud-platform"
    ]
    # Usar a Workload Identity
    workload_metadata_config {
      mode = "GKE_METADATA"
    }
  }
}

resource "google_service_account_iam_binding" "workload_identity_binding" {
  service_account_id = google_service_account.vehicles_sa.name
  role               = "roles/iam.workloadIdentityUser"

  members = [
    "serviceAccount:${var.project_id}.svc.id.goog[vehicles-prod/vehicles-workload-sa]"
  ]
}
