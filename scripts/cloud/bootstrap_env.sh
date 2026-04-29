#!/bin/bash
# 1_bootstrap_env.sh
set -e

echo "Preparing the environment for Google Cloud..."

# Detect OS
OS_TYPE="linux"
if [[ "$OSTYPE" == "darwin"* ]]; then
    OS_TYPE="darwin"
fi

# Download the Proxy (only if it doesn't exist in the current folder)
if [ ! -f "cloud-sql-proxy" ]; then
    echo "Downloading the Cloud SQL Proxy for $OS_TYPE..."
    if [ "$OS_TYPE" == "darwin" ]; then
        curl -o cloud-sql-proxy https://storage.googleapis.com/cloud-sql-connectors/cloud-sql-proxy/v2.13.0/cloud-sql-proxy.darwin.amd64
    else
        curl -o cloud-sql-proxy https://storage.googleapis.com/cloud-sql-connectors/cloud-sql-proxy/v2.13.0/cloud-sql-proxy.linux.amd64
    fi
    chmod +x cloud-sql-proxy
else
    echo "Cloud SQL Proxy already exists in the folder."
fi

echo "Please ensure you have gcloud and kubectl installed."
echo "Run 'gcloud auth login' and 'gcloud auth application-default login' if you haven't already."
echo "Base environment ready!"