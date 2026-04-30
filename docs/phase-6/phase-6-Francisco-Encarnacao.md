# Phase 6 - Francisco Encarnação

## Use Cases

- **Use Case 5 - Distributed Caching**: A user wants to retrieve car listings and market data quickly, even under high traffic, without overloading the underlying databases on every request.
- **Use Case 8 - Infrastructure Reproducibility**: A developer wants to provision or recreate the entire GCP infrastructure (GKE cluster, databases, networking) in a consistent and automated way, without manual steps in the GCP console.

## Requirements

### Functional Requirements

1. **Distributed Caching with Redis**: Integrate GCP Memorystore (Redis) as a cache-aside layer for high-read services such as listings and search, reducing database load and improving response times under traffic spikes.
2. **Infrastructure as Code with Terraform**: Define all GCP infrastructure (GKE cluster, Memorystore, Cloud SQL, networking, IAM) as Terraform code, versioned in GitHub, so that the environment can be reproduced, modified, and torn down reliably without manual steps.

### Non-Functional Requirements

- **Low Latency under Load**: Cached responses for listings and search queries must be served within acceptable latency thresholds even under peak traffic, without hitting the database on every request.
- **Infrastructure Reproducibility**: The entire GCP environment must be expressible as versioned Terraform code, allowing the infrastructure to be recreated identically in any project or region.

## Technical Architecture

To meet the requirements, the technical architecture of the platform will be designed as follows:

1. **Distributed Caching with Redis (Cache-Aside Pattern)**: GCP Memorystore (Redis) will be used as a cache layer for read-heavy services such as listings and search. The cache-aside pattern will be applied: the service first checks Redis for a cached result; on a miss, it queries the database and writes the result back to Redis with a TTL. This reduces database load and improves response times significantly under high traffic. Redis is already used in the chat service for Pub/Sub; this extends its role to caching across other services.
2. **Infrastructure as Code with Terraform**: All GCP resources — including the GKE cluster, node pools, GCP Memorystore instances, Cloud SQL databases, VPC networking, and IAM service accounts — will be defined as Terraform configuration files and versioned in the GitHub repository. This ensures the infrastructure is reproducible, reviewable via pull requests, and not dependent on manual console operations.

## Deployment Plan

The deployment plan for these specific components will involve the following steps:

1. **Deployment of Redis Caching Layer**: Provision a GCP Memorystore (Redis) instance and integrate the cache-aside pattern into the listings and search services. This involves writing the cache read/write logic, configuring TTL values per resource type, and validating cache hit rates under load.
2. **Terraform Infrastructure Codification**: Write Terraform modules for all existing GCP resources (GKE, Memorystore, Cloud SQL, VPC, IAM). Run terraform plan against the live environment to confirm parity, then establish the Terraform state backend in GCP Cloud Storage and integrate Terraform into the CI/CD pipeline for infrastructure changes.
3. **Testing and Validation**: Conduct thorough testing of the caching layer and validate cache hit rates and system performance under simulated load. Run Terraform deployments in a staging environment to ensure full infrastructure reproducibility without manual intervention.
4. **Deployment**: Deploy the updated platform with the new distributed caching capabilities to the production environment and transition all infrastructure management to the newly established Terraform pipelines. Monitor the deployment for any issues and ensure all features are working as expected.