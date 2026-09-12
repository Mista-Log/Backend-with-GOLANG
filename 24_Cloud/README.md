# Go for Beginners — Module 24: Cloud

## Contents

1. **[24-cloud.md](./24-cloud.md)** — Docker (multi-stage builds, why
   distroless images pair with `CGO_ENABLED=0`), Docker Compose,
   Kubernetes (Deployments, Services, and why Module 19's `/healthz`/
   `/readyz` design is exactly the contract it expects), Nginx, Terraform
   and infrastructure as code, a concept-mapping table across AWS/GCP/
   Azure, GitHub Actions and CI/CD, and monitoring with Prometheus and
   Grafana. Diagrams throughout.

2. **[cloud-deployment/](./cloud-deployment)** — One small Go service
   deployed through every layer: a multi-stage `Dockerfile`, a
   `docker-compose.yml` running the app behind Nginx with Prometheus and
   Grafana wired in (runnable right now, no cloud account needed),
   Kubernetes manifests with health probes matching Module 19's contract
   exactly, a Terraform sketch for AWS ECS Fargate, and a GitHub Actions
   CI/CD pipeline.

## Setup

```bash
cd cloud-deployment
docker compose up --build
```

Requires Docker — this is the one module in the course where that's a
prerequisite, since containers and orchestration are the subject matter
itself. Everything else (Kubernetes, Terraform, GitHub Actions) is
included as real, correct configuration you can apply against a real
cluster/cloud account/repo when you have one available, without needing
one just to read and understand this module.

*Note: this module builds directly on Module 19 (the health-check and
metrics contract this deployment exists to serve) and Module 14
(`go test -race`, run in the CI/CD pipeline exactly as it would be run
locally).*
