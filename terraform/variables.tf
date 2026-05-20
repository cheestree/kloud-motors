variable "project_id" { default = "cn-project-491618" }
variable "region" { default = "europe-central2" }
variable "db_password" { sensitive = true }

variable "databases" {
  type        = list(string)
  description = "List of databases to create"
}

variable "artifact_registry_repo" {
  type        = string
  description = "Name of the Artifact Registry repository"
}

variable "backup_bucket_name" {
  type        = string
  description = "Name of the GCS bucket used to store database and dataset backups"
}

variable "pubsub_topics" {
  type        = list(string)
  description = "List of Pub/Sub topics to create"
}
