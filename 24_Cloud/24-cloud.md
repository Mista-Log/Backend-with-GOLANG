# 24. Cloud

Every module before this ran on your own machine. This module covers
packaging a Go service and actually running it in the real world: as a
container, orchestrated across many machines, behind a proxy, on a cloud
provider, deployed automatically, and watched continuously once it's live.

---

## Docker

A **container** packages your app with everything it needs to run,
isolated from the host machine's own environment — the same binary
behaves identically on your laptop, in CI, and in production.

```dockerfile
# Multi-stage build: stage 1 compiles, stage 2 ships ONLY the binary —
# the final image never contains the Go toolchain, source code, or
# build-time dependencies at all.
FROM golang:1.23 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o server ./cmd/server

FROM gcr.io/distroless/static-debian12
COPY --from=builder /app/server /server
EXPOSE 8080
ENTRYPOINT ["/server"]
```

```
┌──────────────────────────────────────────────────────────┐
│   golang:1.23 image:  ~800MB (full Go toolchain + OS)                │
│   distroless/static:   ~2MB base + your static binary                    │
│                                                                              │
│   CGO_ENABLED=0 produces a STATICALLY linked binary — no external          │
│   C library dependencies at all, which is EXACTLY what lets it run              │
│   on a "distroless" image with no shell, no package manager, nothing              │
│   but the binary itself. Smaller image, smaller attack surface.                       │
└──────────────────────────────────────────────────────────┘
```

```bash
docker build -t myapp:latest .
docker run -p 8080:8080 myapp:latest
```

---

## Docker Compose

Defines and runs **multiple** containers together as one unit — your app,
a database, a cache, a monitoring stack — with one command, and a shared
network so they can reach each other by service name.

```yaml
services:
  app:
    build: .
    ports: ["8080:8080"]
    environment:
      - LOG_LEVEL=info
    depends_on: [prometheus]

  prometheus:
    image: prom/prometheus:latest
    ports: ["9090:9090"]
    volumes: ["./monitoring/prometheus.yml:/etc/prometheus/prometheus.yml"]
```

```bash
docker compose up      # starts everything, in dependency order
docker compose down     # tears it all down
```

```
┌────────────────────────────────────────────────────┐
│   Inside the compose network, "app" reaches "prometheus" by          │
│   just that NAME — prometheus:9090 — no IP addresses, no manual          │
│   networking config. Docker Compose's built-in DNS resolves service        │
│   names automatically. This is a small, local taste of Service              │
│   Discovery (Module 20) — the same underlying need, solved simply               │
│   for a single-host development setup.                                            │
└────────────────────────────────────────────────────┘
```

---

## Kubernetes

The dominant system for running containers **at scale**, across many
machines — it schedules containers onto available nodes, restarts crashed
ones, and (via a Service) load-balances traffic across every healthy
replica.

```yaml
# deployment.yaml — WHAT to run, and how many copies
apiVersion: apps/v1
kind: Deployment
metadata:
  name: myapp
spec:
  replicas: 3
  selector:
    matchLabels: {app: myapp}
  template:
    metadata:
      labels: {app: myapp}
    spec:
      containers:
        - name: myapp
          image: myapp:latest
          ports: [{containerPort: 8080}]
          livenessProbe:
            httpGet: {path: /healthz, port: 8080}   # Module 19's liveness check
          readinessProbe:
            httpGet: {path: /readyz, port: 8080}      # Module 19's readiness check
          resources:
            requests: {cpu: "100m", memory: "64Mi"}
            limits: {cpu: "500m", memory: "256Mi"}
---
# service.yaml — HOW to reach those pods, as one stable network identity
apiVersion: v1
kind: Service
metadata: {name: myapp}
spec:
  selector: {app: myapp}
  ports: [{port: 80, targetPort: 8080}]
```

```
┌──────────────────────────────────────────────────────────┐
│   Deployment(replicas: 3)  ──▶  Pod 1, Pod 2, Pod 3  (each a running        │
│                                    copy of your container)                     │
│                                                                                    │
│   Service  ──▶  ONE stable DNS name/IP, load-balancing across WHICHEVER              │
│                    pods are currently marked READY by their readinessProbe               │
│                                                                                                  │
│   A pod failing its livenessProbe repeatedly  ──▶  Kubernetes RESTARTS it                          │
│   A pod failing its readinessProbe             ──▶  Kubernetes STOPS ROUTING                          │
│                                                        traffic to it, without                            │
│                                                        restarting — exactly                                 │
│                                                        Module 19's distinction,                                │
│                                                        now actually acted on                                     │
└──────────────────────────────────────────────────────────┘
```

Module 19's `/healthz` and `/readyz` design wasn't hypothetical — this is
precisely the contract Kubernetes expects, unmodified.

---

## Nginx

A reverse proxy sitting in front of your application(s) — TLS termination,
load balancing across multiple backend instances, request buffering, and
static asset serving, all without your Go code needing to handle any of it.

```nginx
upstream myapp {
    server app1:8080;
    server app2:8080;
}

server {
    listen 443 ssl;
    server_name api.example.com;
    ssl_certificate     /etc/nginx/certs/fullchain.pem;
    ssl_certificate_key /etc/nginx/certs/privkey.pem;

    location / {
        proxy_pass http://myapp;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;      # your Go app sees the REAL client IP here
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

```
┌────────────────────────────────────────────────────┐
│   Browser ──HTTPS──▶ Nginx ──HTTP (internal network)──▶ your Go app        │
│                                                                                 │
│   Nginx terminates TLS ONCE, at the edge — your Go service itself                 │
│   never needs to handle certificates at all, and can be replicated                   │
│   behind Nginx's load balancing with zero code changes.                                 │
└────────────────────────────────────────────────────┘
```

---
## Terraform

**Infrastructure as Code** — declaring your cloud infrastructure (servers,
networks, databases) in files, version-controlled exactly like application
code, instead of clicking through a cloud console by hand.

```hcl
resource "aws_ecs_service" "myapp" {
  name            = "myapp"
  cluster         = aws_ecs_cluster.main.id
  task_definition = aws_ecs_task_definition.myapp.arn
  desired_count   = 3

  load_balancer {
    target_group_arn = aws_lb_target_group.myapp.arn
    container_name   = "myapp"
    container_port   = 8080
  }
}
```

```bash
terraform plan     # shows EXACTLY what would change, before touching anything
terraform apply     # actually applies it
```

```
┌──────────────────────────────────────────────────────────┐
│   terraform plan is Module 19's graceful-shutdown philosophy,           │
│   applied to infrastructure: SHOW what's about to happen, in                │
│   detail, before committing to it — exactly like reviewing a                    │
│   database migration (Module 17) before running it against production.            │
│                                                                                        │
│   Terraform tracks STATE (what it believes currently exists) and                        │
│   reconciles your declared configuration against that state — changing                     │
│   the .tf file and re-applying updates ONLY what actually changed.                            │
└──────────────────────────────────────────────────────────┘
```

---

## AWS / GCP / Azure

The three dominant cloud providers — different names for largely
equivalent underlying concepts. Knowing the mapping matters more than
memorizing any one provider's exact console:

```
┌──────────────────────────────────────────────────────────┐
│   CONCEPT                  AWS              GCP                AZURE       │
│  ──────────────────────────────────────────────────────────  │
│   Virtual machine           EC2               Compute Engine     VMs           │
│   Managed containers          ECS/Fargate        Cloud Run           Container       │
│                                                                        Instances      │
│   Kubernetes                   EKS                 GKE                  AKS             │
│   Object storage                 S3                   Cloud Storage       Blob Storage      │
│   Managed relational DB            RDS                  Cloud SQL             Azure SQL        │
│   Serverless functions               Lambda               Cloud Functions       Functions         │
│   Secrets management                    Secrets Manager     Secret Manager        Key Vault           │
│   CDN                                      CloudFront          Cloud CDN             Azure CDN             │
└──────────────────────────────────────────────────────────┘
```

A Go binary deployed as a container behaves nearly identically across all
three — the meaningful differences are in each provider's surrounding
services (IAM/permissions models, networking setup, pricing), not in how
your Go code itself runs.

---
## GitHub Actions & CI/CD

**Continuous Integration** runs your tests (and linting, and builds)
automatically on every push — catching problems before they reach anyone
else. **Continuous Deployment** takes it further, automatically shipping
every change that passes CI to production (or staging).

```yaml
# .github/workflows/ci-cd.yml
name: CI/CD
on:
  push:
    branches: [main]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: {go-version: '1.23'}
      - run: go build ./...
      - run: go vet ./...
      - run: go test -race ./...          # Module 12's race detector, run on EVERY push

  deploy:
    needs: test                              # only runs if `test` succeeded
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - run: docker build -t myapp:${{ github.sha }} .
      - run: docker push myapp:${{ github.sha }}
      - run: kubectl set image deployment/myapp myapp=myapp:${{ github.sha }}
```

```
┌──────────────────────────────────────────────────────────┐
│   push to main                                                       │
│        │                                                                │
│        ▼                                                                  │
│   test job:  build → vet → test -race                                       │
│        │  ANY step fails → PIPELINE STOPS, deploy job never runs                │
│        ▼  all pass                                                                 │
│   deploy job:  build image → push → update the Kubernetes Deployment                    │
│                                                                                              │
│   The `needs: test` line is the whole safety mechanism — nothing EVER                          │
│   reaches deployment without passing the exact same test suite you run                            │
│   locally, automatically, on every single change.                                                    │
└──────────────────────────────────────────────────────────┘
```

---

## Monitoring

Once a service is live, you need **continuous visibility** into whether
it's actually healthy — this is Module 19's Observability section, now
wired into real infrastructure instead of just exposed on an endpoint
nobody's watching yet.

```
┌──────────────────────────────────────────────────────────┐
│   Your Go app exposes /metrics (Module 19, prometheus/client_golang)   │
│        │                                                                   │
│        ▼                                                                     │
│   Prometheus SCRAPES it periodically, stores the time series                    │
│        │                                                                           │
│        ▼                                                                             │
│   Grafana QUERIES Prometheus, renders dashboards and fires alerts                       │
└──────────────────────────────────────────────────────────┘
```

---

## Prometheus

The metrics collection and storage system — it **pulls** metrics from
your service on a schedule (rather than your service pushing them out),
which keeps the collector in full control of scrape frequency and load.

```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'myapp'
    scrape_interval: 15s
    static_configs:
      - targets: ['app:8080']    # Docker Compose's service-name DNS, from earlier
```

```
┌────────────────────────────────────────────────────┐
│   PromQL — Prometheus's query language — answers questions like:              │
│                                                                                    │
│   rate(http_requests_total[5m])          → requests per second, averaged             │
│                                              over the last 5 minutes                     │
│   histogram_quantile(0.95, ...)            → the 95th percentile request latency          │
│   up == 0                                    → which targets are currently DOWN              │
└────────────────────────────────────────────────────┘
```

---

## Grafana

The visualization layer on top of Prometheus (and other data sources) —
dashboards, graphs, and **alerts** (notify someone the moment a metric
crosses a threshold, rather than waiting for a customer to report it).

```
┌──────────────────────────────────────────────────────────┐
│   A dashboard panel querying Prometheus:                              │
│                                                                            │
│   rate(http_requests_total{status=~"5.."}[5m])                              │
│                                                                                  │
│   → graphs the ERROR rate over time — an alert rule on this exact query          │
│      firing when it exceeds, say, 1% for 5 straight minutes is a genuinely           │
│      production-grade "something is wrong" signal, built entirely from                  │
│      the SAME /metrics endpoint Module 19 already wired into the app.                       │
└──────────────────────────────────────────────────────────┘
```

---

Onto the project — a complete deployment for a small Go service: a
multi-stage Dockerfile, a Docker Compose stack (app + Nginx + Prometheus +
Grafana) you can run locally right now, Kubernetes manifests, a Terraform
sketch for AWS, and a GitHub Actions pipeline — every piece from this
guide, wired together around one real, runnable Go service.
