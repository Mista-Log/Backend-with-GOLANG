# Reference Project — Cloud Deployment

One small Go service (`app/`), deployed through every layer this module's
guide covers: Docker, Docker Compose, Kubernetes, Nginx, Terraform,
CI/CD, Prometheus, and Grafana.

## Run the Whole Stack Locally

```bash
cd cloud-deployment
docker compose up --build
```

```
┌──────────────────────────────────────────────────────────┐
│   curl http://localhost:8080/          → Nginx → app                  │
│   open http://localhost:9090            → Prometheus (scraping app)      │
│   open http://localhost:3000            → Grafana (admin/admin,             │
│                                              Prometheus datasource pre-        │
│                                              provisioned automatically)           │
└──────────────────────────────────────────────────────────┘
```

Generate a little traffic, then watch it show up in Prometheus:
```bash
for i in $(seq 1 50); do curl -s http://localhost:8080/ > /dev/null; done
# In Prometheus (localhost:9090), query: http_requests_total
```

## Deploying to Kubernetes (if you have a cluster available)

```bash
docker build -t cloudapp:latest ./app
kubectl apply -f k8s/deployment.yaml
kubectl apply -f k8s/hpa.yaml
kubectl get pods -l app=cloudapp     # watch 3 replicas come up
kubectl port-forward svc/cloudapp 8080:80
```

Kill a pod on purpose and watch Kubernetes replace it automatically:
```bash
kubectl delete pod -l app=cloudapp --field-selector status.phase=Running -o name | head -1 | xargs kubectl delete
kubectl get pods -l app=cloudapp -w   # a new one appears within seconds
```

## What Each Piece Maps to in the Guide

| File | Guide section |
|---|---|
| `app/Dockerfile` | Docker — multi-stage build, distroless final image |
| `docker-compose.yml` | Docker Compose — app + Nginx + Prometheus + Grafana, one command |
| `nginx/nginx.conf` | Nginx — reverse proxy, header forwarding |
| `k8s/deployment.yaml` | Kubernetes — replicas, liveness/readiness probes matching Module 19 exactly |
| `k8s/hpa.yaml` | Kubernetes — autoscaling based on real CPU load |
| `terraform/main.tf` | Terraform — declarative AWS ECS Fargate sketch |
| `.github/workflows/ci-cd.yml` | GitHub Actions / CI/CD — test-then-deploy pipeline |
| `monitoring/prometheus.yml` | Prometheus — scrape config |
| `monitoring/grafana/datasource.yml` | Grafana — auto-provisioned datasource |

## Case Study: Why the Liveness and Readiness Probes Point at Different Paths

`k8s/deployment.yaml`'s `livenessProbe` checks `/healthz`; its
`readinessProbe` checks `/readyz` — deliberately the same two, differently
named endpoints Module 19 built specifically for this contract. In this
demo app both always return 200 (there's no real dependency to check), but
the **shape** is what matters: if `app/main.go` later gained a real
database connection, `/readyz` is exactly where you'd add a ping-the-
database check, and Kubernetes would automatically stop routing traffic to
any replica whose database connection dropped — without ever restarting a
perfectly healthy process over a problem restarting it can't fix.

## Try It Yourself
- Add a Grafana dashboard JSON (provisioned the same way as the
  datasource) graphing `rate(http_requests_total[1m])`
- Make `/readyz` actually fail sometimes (a toggle via an env var) and
  watch `kubectl get pods` show a pod as `Running` but `0/1 Ready` —
  proving Kubernetes really does treat the two probes differently
- Fill in `terraform/main.tf`'s missing VPC/subnet/load-balancer resources
  for a genuinely `terraform apply`-able configuration (a substantial,
  realistic exercise in its own right)
