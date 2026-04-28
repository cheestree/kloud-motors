# Phase 6 - Non-Functional Requirements and Technical Architecture

## Use Cases

- **Use Case 1 - Privacy and Data Protection**: A user wants to ensure that their personal information and data are protected while using the platform.
- **Use Case 2 - Cluster Monitoring**: A user wants to monitor the health and performance of the containerized applications in the Kubernetes cluster and receive alerts in case of issues.
- **Use Case 3 - Data Backup and Recovery**: A user wants to ensure that the dataset and application data are regularly backed up and can be recovered in case of data loss or system failure.
- **Use Case 4 - Automated Deployment**: A developer wants to automatically deploy updated services to the GCP Kubernetes cluster whenever changes are merged into the main branch for example, ensuring that the latest features and fixes are available to users without manual intervention.

## Requirements

### Functional Requirements

- **User Authentication**: Allow the use of secure authentication methods (e.g. OAuth, JWT) to protect user data and ensure secure access to the platform via a Cloud Identity Provider (e.g. AWS Cognito, Auth0).

- **Cluster Monitoring**: Implement monitoring and alerting mechanisms to track the health and performance of the containerized applications in the Kubernetes cluster via tools like Prometheus and Grafana (but more so Prometheus).

- **Data Backup and Recovery**: Establish a robust backup and recovery strategy for the dataset and application data to prevent data loss and ensure business continuity. This may involve regular backups to cloud storage services (e.g. AWS S3, Google Cloud Storage) and implementing disaster recovery plans.

- **Automated CI/CD Pipeline**: Implement a continuous integration and continuous deployment pipeline so that every merge to the main branch automatically builds, tests, and deploys the affected microservice(s) to the GKE cluster on GCP, ensuring fast and reliable delivery of changes.

### Non-Functional Requirements

- **Zero-Downtime Deployments**: Deployments must not interrupt active users. Rolling update strategies will be used to gradually replace old pods with new ones.
- **Deployment Auditability**: Every deployment must be traceable to a specific commit and pull request, providing a clear audit trail of what was deployed, when, and by whom.
- **Environment Isolation**: The pipeline will distinguish between branches, deploying feature branches to a staging environment and the main branch to production.
- **Least-Privilege Access**: The CI/CD pipeline will authenticate with GCP using a dedicated service account with the minimum permissions required (Workload Identity Federation), avoiding long-lived credential keys.

## Technical Architecture

To meet the requirements, the technical architecture of the platform will be designed as follows:

- **Removal of the Authentication Service**: The authentication service will be removed from the architecture, and instead, a Cloud Identity Provider (e.g. AWS Cognito, Auth0) will be integrated to handle user authentication and data protection. Data in regards to user IDs and other relevant information will be stored in the database, but the authentication process will be managed by the Cloud Identity Provider.

- **Cluster Monitoring**: The Kubernetes cluster will be monitored using Prometheus for collecting metrics and Grafana for visualizing the data. Alerts will be configured to notify administrators of any issues or performance degradation in the cluster, requiring changes of the architecture to include a monitoring stack or endpoints for monitoring.

- **Data Backup and Recovery**: A backup strategy will be implemented to regularly back up the dataset and application data to a cloud storage service (e.g. AWS S3, Google Cloud Storage). This will involve setting up automated backup schedules with jobs and ensuring that the backup data is securely stored and can be easily recovered in case of data loss or system failure.

- **CI/CD Pipeline with GitOps (GitHub Actions → GKE)**: A GitHub Actions workflow will be configured to trigger on every push or merge to the main branch. The pipeline will build a Docker image for the affected microservice, push it to Google Artifact Registry, and apply the updated Kubernetes manifests to the GKE cluster. Secrets will be managed via GitHub Secrets and GCP Secret Manager, and authentication will use Workload Identity Federation to avoid storing long-lived service account keys. The pipeline stages are: **lint → test → build → push → deploy**.

## Deployment Plan

The deployment plan will involve the following steps:

1. **Integration of Cloud Identity Provider**: Integrate the chosen Cloud Identity Provider (e.g. AWS Cognito, Auth0) into the platform to handle user authentication and data protection. This will involve configuring the authentication flow and ensuring that user data is securely stored in the database.
2. **Setup of Cluster Monitoring**: Implement Prometheus and Grafana for monitoring the Kubernetes cluster. This will involve configuring Prometheus to collect metrics from the cluster and setting up Grafana dashboards for visualizing the data. Alerts will be configured to notify administrators of any issues.
3. **Implementation of Data Backup and Recovery**: Establish a backup strategy for the dataset and application data. This will involve setting up automated backup schedules to a cloud storage service (e.g. AWS S3, Google Cloud Storage) and ensuring that the backup data is securely stored and can be easily recovered in case of data loss or system failure.
4. **Setup of CI/CD Pipeline**: Configure GitHub Actions workflows for each microservice repository. This involves setting up Workload Identity Federation between GitHub and GCP, creating a dedicated GCP service account with permissions scoped to Artifact Registry and GKE, writing the workflow YAML (lint → test → build → push → deploy), and validating the pipeline end-to-end with a test deployment to the staging environment.
5. **Testing and Validation**: Conduct thorough testing of the integrated Cloud Identity Provider, cluster monitoring setup, data backup and recovery processes, and the CI/CD pipeline to ensure that they are functioning correctly and meeting the requirements.
6. **Deployment**: Deploy the updated platform with the integrated Cloud Identity Provider, cluster monitoring, data backup and recovery mechanisms, and automated CI/CD pipeline to the production environment. Monitor the deployment for any issues and ensure that the new features are working as expected.