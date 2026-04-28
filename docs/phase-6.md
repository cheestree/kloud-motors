# Phase 6 - Non-Functional Requirements and Technical Architecture

## Use Cases

- **Use Case 1 - Privacy and Data Protection**: A user wants to ensure that their personal information and data are protected while using the platform.
- **Use Case 2 - Cluster Monitoring**: A user wants to monitor the health and performance of the containerized applications in the Kubernetes cluster and receive alerts in case of issues.
- **Use Case 3 - Data Backup and Recovery**: A user wants to ensure that the dataset and application data are regularly backed up and can be recovered in case of data loss or system

## Requirements

### Functional Requirements

- **User Authentication**: Allow the use of secure authentication methods (e.g. OAuth, JWT) to protect user data and ensure secure access to the platform via a Cloud Identity Provider (e.g. AWS Cognito, Auth0).

- **Cluster Monitoring**: Implement monitoring and alerting mechanisms to track the health and performance of the containerized applications in the Kubernetes cluster via tools like Prometheus and Grafana (but more so Prometheus).

- **Data Backup and Recovery**: Establish a robust backup and recovery strategy for the dataset and application data to prevent data loss and ensure business continuity. This may involve regular backups to cloud storage services (e.g. AWS S3, Google Cloud Storage) and implementing disaster recovery plans.

## Technical Architecture

To meet the requirements, the technical architecture of the platform will be designed as follows:

- **Removal of the Authentication Service**: The authentication service will be removed from the architecture, and instead, a Cloud Identity Provider (e.g. AWS Cognito, Auth0) will be integrated to handle user authentication and data protection. Data in regards to user IDs and other relevant information will be stored in the database, but the authentication process will be managed by the Cloud Identity Provider.

- **Cluster Monitoring**: The Kubernetes cluster will be monitored using Prometheus for collecting metrics and Grafana for visualizing the data. Alerts will be configured to notify administrators of any issues or performance degradation in the cluster, requiring changes of the architecture to include a monitoring stack or endpoints for monitoring.

- **Data Backup and Recovery**: A backup strategy will be implemented to regularly back up the dataset and application data to a cloud storage service (e.g. AWS S3, Google Cloud Storage). This will involve setting up automated backup schedules with jobs and ensuring that the backup data is securely stored and can be easily recovered in case of data loss or system failure.

## Deployment Plan

The deployment plan will involve the following steps:

1. **Integration of Cloud Identity Provider**: Integrate the chosen Cloud Identity Provider (e.g. AWS Cognito, Auth0) into the platform to handle user authentication and data protection. This will involve configuring the authentication flow and ensuring that user data is securely stored in the database.
2. **Setup of Cluster Monitoring**: Implement Prometheus and Grafana for monitoring the Kubernetes cluster. This will involve configuring Prometheus to collect metrics from the cluster and setting up Grafana dashboards for visualizing the data. Alerts will be configured to notify administrators of any issues.
3. **Implementation of Data Backup and Recovery**: Establish a backup strategy for the dataset and application data. This will involve setting up automated backup schedules to a cloud storage service (e.g. AWS S3, Google Cloud Storage) and ensuring that the backup data is securely stored and can be easily recovered in case of data loss or system failure.
4. **Testing and Validation**: Conduct thorough testing of the integrated Cloud Identity Provider, cluster monitoring setup, and data backup and recovery processes to ensure that they are functioning correctly and meeting the requirements.
5. **Deployment**: Deploy the updated platform with the integrated Cloud Identity Provider, cluster monitoring, and data backup and recovery mechanisms to the production environment. Monitor the deployment for any issues and ensure that the new features are working as expected.