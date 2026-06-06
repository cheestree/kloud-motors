# AGENTS.md

Guidance for AI coding agents working in this repository.

## Project Overview

This repository contains a cloud computing project built around Go microservices, PostgreSQL databases, a REST gateway, gRPC services, Docker Compose for local execution, Kubernetes manifests for deployment, and Terraform for cloud infrastructure.

Key areas:

- `code/services/`: Go microservices, protobuf generation scripts, and Go module files.
- `api/API.yaml`: REST API contract.
- `scripts/local/`: local setup, startup, seeding, and integration-test scripts.
- `scripts/cloud/`: cloud, Kubernetes, database, backup, and Terraform helper scripts.
- `deploy/k8s/`: Kubernetes manifests and kustomization files.
- `terraform/`: infrastructure configuration.
- `g16_tests/`: smoke, stress, cloudy-day, Locust, and report artifacts.
- `docs/`: project phase documentation.

## Working Rules

- Prefer existing scripts and patterns over introducing new tooling.
- Keep changes scoped to the requested behavior.
- Do not commit secrets, service-account files, `.env` files, kubeconfigs, logs, generated binaries, or local machine artifacts.
- Treat `gcp-sa-key.json`, `service-account.json`, `.env`, `.env.secrets`, `.env.variables`, `deploy/k8s/kubeconfig`, and `proxy.log` as sensitive/local files.
- Avoid modifying generated or environment-specific files unless the task explicitly requires it.
- Before editing infrastructure or deployment files, inspect related scripts and manifests so changes stay consistent.

## Common Commands

Prepare local dataset artifacts:

```bash
./scripts/local/prepare.sh
```

Start local services and databases:

```bash
./scripts/local/start.sh
```

Seed listing data:

```bash
./scripts/local/seed.sh
```

Run local integration tests:

```bash
./scripts/local/test-integration.sh
```

Generate protobuf code from the services directory:

```bash
cd code/services
./gen_proto.sh
```

Manage Kubernetes deployment:

```bash
./scripts/cloud/k8s.sh status
./scripts/cloud/k8s.sh up
./scripts/cloud/k8s.sh up --with-ingress
./scripts/cloud/k8s.sh down
```

## Validation

Choose the narrowest validation that covers the change:

- For Go service changes, run the relevant Go tests from `code/services`.
- For API contract changes, update `api/API.yaml` and check affected gateway/service code.
- For Docker or local runtime changes, run the affected `scripts/local/*` command.
- For Kubernetes changes, validate the related manifest under `deploy/k8s/` and use `./scripts/cloud/k8s.sh status` when a cluster context is available.
- For Terraform changes, run validation from `terraform/` when Terraform and cloud credentials are available.
- For test-only changes, run the specific smoke, stress, Locust, or shell test that was changed.

If credentials, external services, Terraform state, Kubernetes context, or datasets are unavailable, state exactly which validation could not be performed.

## Code Style

- Follow the existing Go module and package organization in `code/services`.
- Keep shell scripts POSIX-oriented where possible, and preserve existing argument conventions.
- Keep YAML indentation and resource names consistent with nearby Kubernetes and workflow files.
- Update documentation when behavior, setup steps, ports, or API routes change.

## API And Ports

The REST gateway defaults to `http://localhost:8080` locally. gRPC and WebSocket ports are documented in `README.md`; check that file before changing service ports or route descriptions.

## Safety Notes

- Do not run destructive cloud, database, Kubernetes, or Terraform commands unless explicitly requested.
- Do not rotate, overwrite, or remove service-account credentials without explicit approval.
- Do not assume cloud credentials or kubeconfig are valid on the current machine.
- Preserve user changes in the worktree; do not revert unrelated files.
