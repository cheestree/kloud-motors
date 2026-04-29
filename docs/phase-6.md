# Phase 6 - Non-Functional Requirements and Technical Architecture

## Use Cases

- **Use Case 1 - Privacy and Data Protection**: A user wants to ensure that their personal information and data are protected while using the platform.
- **Use Case 2 - Cluster Monitoring**: A user wants to monitor the health and performance of the containerized applications in the Kubernetes cluster and receive alerts in case of issues.
- **Use Case 3 - Data Backup and Recovery**: A user wants to ensure that the dataset and application data are regularly backed up and can be recovered in case of data loss or system failure.
- **Use Case 4 - Automated Deployment**: A developer wants to automatically deploy updated services to the GCP Kubernetes cluster whenever changes are merged into the main branch for example, ensuring that the latest features and fixes are available to users without manual intervention.
- **Use Case 5 - Distributed Caching**: A user wants to retrieve car listings and market data quickly, even under high traffic, without overloading the underlying databases on every request.
- **Use Case 6 - Service Resilience**: A user wants to continue using the platform normally even when a downstream microservice (e.g. listings or auctions) is temporarily unavailable or slow, without experiencing cascading failures.
- **Use Case 7 - Request Tracing**: A developer wants to trace a slow or failing request across multiple microservices to quickly identify which service is the bottleneck or source of the error.
- **Use Case 8 - Infrastructure Reproducibility**: A developer wants to provision or recreate the entire GCP infrastructure (GKE cluster, databases, networking) in a consistent and automated way, without manual steps in the GCP console.

## Requirements

### Functional Requirements

- **User Authentication**: Allow the use of secure authentication methods (e.g. OAuth, JWT) to protect user data and ensure secure access to the platform via a Cloud Identity Provider (e.g. AWS Cognito, Auth0).

- **Cluster Monitoring**: Implement monitoring and alerting mechanisms to track the health and performance of the containerized applications in the Kubernetes cluster via tools like Prometheus and Grafana (but more so Prometheus).

- **Data Backup and Recovery**: Establish a robust backup and recovery strategy for the dataset and application data to prevent data loss and ensure business continuity. This may involve regular backups to cloud storage services (e.g. AWS S3, Google Cloud Storage) and implementing disaster recovery plans.

- **Automated CI/CD Pipeline**: Implement a continuous integration and continuous deployment pipeline so that every merge to the main branch automatically builds, tests, and deploys the affected microservice(s) to the GKE cluster on GCP, ensuring fast and reliable delivery of changes.

- **Distributed Caching with Redis**: Integrate GCP Memorystore (Redis) as a cache-aside layer for high-read services such as listings and search, reducing database load and improving response times under traffic spikes.

- **Circuit Breaker and Retry**: Implement circuit breaker and retry patterns on inter-service gRPC calls so that a failing or slow downstream service does not cause cascading failures across the platform. Services must degrade gracefully when dependencies are unavailable.

- **Distributed Tracing**: Instrument all microservices with OpenTelemetry to emit trace data, enabling end-to-end visibility of requests as they flow across services. Traces will be collected and visualised via GCP Cloud Trace.

- **Infrastructure as Code with Terraform**: Define all GCP infrastructure (GKE cluster, Memorystore, Cloud SQL, networking, IAM) as Terraform code, versioned in GitHub, so that the environment can be reproduced, modified, and torn down reliably without manual steps.

### Non-Functional Requirements

- **Zero-Downtime Deployments**: Deployments must not interrupt active users. Rolling update strategies will be used to gradually replace old pods with new ones.
- **Deployment Auditability**: Every deployment must be traceable to a specific commit and pull request, providing a clear audit trail of what was deployed, when, and by whom.
- **Environment Isolation**: The pipeline will distinguish between branches, deploying feature branches to a staging environment and the main branch to production.
- **Least-Privilege Access**: The CI/CD pipeline will authenticate with GCP using a dedicated service account with the minimum permissions required (Workload Identity Federation), avoiding long-lived credential keys.
- **Low Latency under Load**: Cached responses for listings and search queries must be served within acceptable latency thresholds even under peak traffic, without hitting the database on every request.
- **Fault Tolerance**: The platform must remain operational and return meaningful responses even when individual microservices are unavailable, by applying circuit breaker thresholds and fallback responses.
- **Observability**: Every request crossing a service boundary must produce a trace span, enabling developers to identify latency bottlenecks and errors across the distributed system without relying solely on logs.
- **Infrastructure Reproducibility**: The entire GCP environment must be expressible as versioned Terraform code, allowing the infrastructure to be recreated identically in any project or region.

## Technical Architecture

To meet the requirements, the technical architecture of the platform will be designed as follows:

- **Removal of the Authentication Service**: The authentication service will be removed from the architecture, and instead, a Cloud Identity Provider (e.g. AWS Cognito, Auth0) will be integrated to handle user authentication and data protection. Data in regards to user IDs and other relevant information will be stored in the database, but the authentication process will be managed by the Cloud Identity Provider.

- **Cluster Monitoring**: The Kubernetes cluster will be monitored using Prometheus for collecting metrics and Grafana for visualizing the data. Alerts will be configured to notify administrators of any issues or performance degradation in the cluster, requiring changes of the architecture to include a monitoring stack or endpoints for monitoring.

- **Data Backup and Recovery**: A backup strategy will be implemented to regularly back up the dataset and application data to a cloud storage service (e.g. AWS S3, Google Cloud Storage). This will involve setting up automated backup schedules with jobs and ensuring that the backup data is securely stored and can be easily recovered in case of data loss or system failure.

- **CI/CD Pipeline with GitOps (GitHub Actions → GKE)**: A GitHub Actions workflow will be configured to trigger on every push or merge to the main branch. The pipeline will build a Docker image for the affected microservice, push it to Google Artifact Registry, and apply the updated Kubernetes manifests to the GKE cluster. Secrets will be managed via GitHub Secrets and GCP Secret Manager, and authentication will use Workload Identity Federation to avoid storing long-lived service account keys. The pipeline stages are: **lint → test → build → push → deploy**.

- **Distributed Caching with Redis (Cache-Aside Pattern)**: GCP Memorystore (Redis) will be used as a cache layer for read-heavy services such as listings and search. The cache-aside pattern will be applied: the service first checks Redis for a cached result; on a miss, it queries the database and writes the result back to Redis with a TTL. This reduces database load and improves response times significantly under high traffic. Redis is already used in the chat service for Pub/Sub; this extends its role to caching across other services.

- **Circuit Breaker and Retry (Inter-Service Resilience)**: All outbound gRPC calls between microservices will be wrapped with a circuit breaker using a library such as `gobreaker`. If a downstream service exceeds a failure threshold, the circuit opens and subsequent calls fail fast with a fallback response, preventing cascading failures. Retries with exponential backoff will be applied to transient errors before the circuit opens.

- **Distributed Tracing with OpenTelemetry and GCP Cloud Trace**: All microservices will be instrumented with the OpenTelemetry Go SDK to emit trace spans for every inbound and outbound request (gRPC and HTTP). Traces will be exported to GCP Cloud Trace, where they can be visualised as end-to-end request timelines across services. This provides developers with the observability needed to diagnose latency and errors in the distributed system.

- **Infrastructure as Code with Terraform**: All GCP resources — including the GKE cluster, node pools, GCP Memorystore instances, Cloud SQL databases, VPC networking, and IAM service accounts — will be defined as Terraform configuration files and versioned in the GitHub repository. This ensures the infrastructure is reproducible, reviewable via pull requests, and not dependent on manual console operations.

## Deployment Plan

The deployment plan will involve the following steps:

1. **Integration of Cloud Identity Provider**: Integrate the chosen Cloud Identity Provider (e.g. AWS Cognito, Auth0) into the platform to handle user authentication and data protection. This will involve configuring the authentication flow and ensuring that user data is securely stored in the database.
2. **Setup of Cluster Monitoring**: Implement Prometheus and Grafana for monitoring the Kubernetes cluster. This will involve configuring Prometheus to collect metrics from the cluster and setting up Grafana dashboards for visualizing the data. Alerts will be configured to notify administrators of any issues.
3. **Implementation of Data Backup and Recovery**: Establish a backup strategy for the dataset and application data. This will involve setting up automated backup schedules to a cloud storage service (e.g. AWS S3, Google Cloud Storage) and ensuring that the backup data is securely stored and can be easily recovered in case of data loss or system failure.
4. **Setup of CI/CD Pipeline**: Configure GitHub Actions workflows for each microservice repository. This involves setting up Workload Identity Federation between GitHub and GCP, creating a dedicated GCP service account with permissions scoped to Artifact Registry and GKE, writing the workflow YAML (lint → test → build → push → deploy), and validating the pipeline end-to-end with a test deployment to the staging environment.
5. **Deployment of Redis Caching Layer**: Provision a GCP Memorystore (Redis) instance and integrate the cache-aside pattern into the listings and search services. This involves writing the cache read/write logic, configuring TTL values per resource type, and validating cache hit rates under load.
6. **Implementation of Circuit Breaker and Retry**: Integrate circuit breaker and retry logic into all inter-service gRPC clients. This involves configuring failure thresholds, backoff strategies, and fallback responses, followed by testing failure scenarios (e.g. killing a downstream service pod) to confirm the circuit opens correctly.
7. **Instrumentation with OpenTelemetry**: Add OpenTelemetry SDK instrumentation to each microservice and configure the trace exporter to send data to GCP Cloud Trace. Validate that end-to-end traces appear correctly in the Cloud Trace UI for representative request flows.
8. **Terraform Infrastructure Codification**: Write Terraform modules for all existing GCP resources (GKE, Memorystore, Cloud SQL, VPC, IAM). Run `terraform plan` against the live environment to confirm parity, then establish the Terraform state backend in GCP Cloud Storage and integrate Terraform into the CI/CD pipeline for infrastructure changes.
9. **Testing and Validation**: Conduct thorough testing of the integrated Cloud Identity Provider, cluster monitoring setup, data backup and recovery processes, CI/CD pipeline, caching layer, circuit breaker behaviour, distributed traces, and Terraform reproducibility to ensure all components are functioning correctly.
10. **Deployment**: Deploy the updated platform with all new non-functional capabilities to the production environment. Monitor the deployment for any issues and ensure all features are working as expected.

## Student Distribution

- **Francisco Encarnação**
  - Use Case 5 - Distributed Caching
  - Use Case 8 - Infrastructure Reproducibility

- **Daniel Carvalho**
  - Use Case 2 - Cluster Monitoring
  - Use Case 6 - Service Resilience

- **Daniel Nunes**
  - Use Case 1 - Privacy and Data Protection
  - Use Case 3 - Data Backup and Recovery

- **Daniel Sousa**
  - Use Case 4 - Automated Deployment
  - Use Case 7 - Request Tracing
