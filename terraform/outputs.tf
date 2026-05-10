output "db_instance_connection_name" {
  value = google_sql_database_instance.db_instance.connection_name
}
output "kubernetes_cluster_name" {
  value = google_container_cluster.primary.name
}