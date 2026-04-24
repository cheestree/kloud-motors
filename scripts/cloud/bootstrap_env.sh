#!/bin/bash
# 1_bootstrap_env.sh
set -e

echo "Preparing the environment for Google Cloud..."

# Download the Proxy (only if it doesn't exist in the current folder)
if [ ! -f "cloud-sql-proxy" ]; then
    echo "Downloading the Cloud SQL Proxy..."
    curl -o cloud-sql-proxy https://storage.googleapis.com/cloud-sql-connectors/cloud-sql-proxy/v2.13.0/cloud-sql-proxy.linux.amd64
    chmod +x cloud-sql-proxy
else
    echo "Cloud SQL Proxy already exists in the folder."
fi

echo "Please ensure you have gcloud and kubectl installed."
echo "Run 'gcloud auth login' and 'gcloud auth application-default login' if you haven't already."
echo "Base environment ready!"