# gforce

**gforce** is an open-source, enterprise-grade Git platform designed from the ground up for Kubernetes. Every repository, user, and access control rule is a Kubernetes Custom Resource — not a database record bolted onto a container. gforce speaks Git natively over HTTP and SSH, exposes a JSON API, and ships a Kubernetes operator that reconciles desired repository state with on-disk reality.

Unlike GitHub-style platforms that run *on* Kubernetes as an afterthought, gforce is built *for* Kubernetes. GitOps workflows, multi-tenant namespace isolation, RBAC via `ClusterRole`/`RoleBinding`, and horizontal scaling are first-class concerns — not plugins.

## Architecture

```
                       ┌─────────────────────────────────────────────┐
                       │                  Kubernetes                  │
                       │                                              │
    kubectl / CI ──────►  Repository CRD  ◄──── gforce operator      │
                       │        │                    │                │
                       │        │  reconcile         │ init bare repo │
                       │        ▼                    ▼                │
                       │   gforce-system ns     PersistentVolume      │
                       └─────────────────────────────────────────────┘
                                │
              ┌─────────────────┼─────────────────┐
              │                 │                 │
              ▼                 ▼                 ▼
       git clone/push     REST API /        Prometheus
       (smart-HTTP)       api/v1/...        /metrics
              │                 │
              │       ┌─────────┴──────────┐
              │       │   gforce server    │
              │       │  ┌─────────────┐  │
              └───────►  │  gitserver  │  │
                      │  └─────────────┘  │
                      │  ┌─────────────┐  │
                      │  │  api router │  │
                      │  └──────┬──────┘  │
                      │         │         │
                      └─────────┼─────────┘
                                │
                      ┌─────────▼─────────┐
                      │    PostgreSQL      │
                      │  users / repos /   │
                      │  ssh_keys tables   │
                      └───────────────────┘
```

## Quickstart (minikube + Helm)

**Prerequisites:** `minikube`, `helm`, `kubectl`

```bash
# Start a local cluster
minikube start --cpus=2 --memory=4096

# Add the gforce Helm repo (once published)
helm repo add gforce https://charts.gforce.dev
helm repo update

# Create a namespace and a secret with required credentials
kubectl create namespace gforce-system
kubectl -n gforce-system create secret generic gforce \
  --from-literal=db-dsn="postgres://gforce:password@gforce-postgresql:5432/gforce?sslmode=disable" \
  --from-literal=jwt-secret="$(openssl rand -hex 32)"

# Install
helm install gforce gforce/gforce \
  --namespace gforce-system \
  --set existingSecret=gforce

# Verify
kubectl -n gforce-system rollout status deployment/gforce
kubectl -n gforce-system port-forward svc/gforce 8080:8080 &
curl http://localhost:8080/healthz
```

## Development Setup

**Prerequisites:** Go 1.22+, Docker, `golangci-lint`, `controller-gen`, `air`

```bash
git clone https://github.com/gforce/gforce.git
cd gforce

# Install dev tools (first time only)
go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.59.1
go install sigs.k8s.io/controller-tools/cmd/controller-gen@v0.15.0
go install github.com/air-verse/air@latest

# Start a local PostgreSQL instance
docker run -d --name gforce-pg \
  -e POSTGRES_DB=gforce \
  -e POSTGRES_USER=gforce \
  -e POSTGRES_PASSWORD=devpassword \
  -p 5432:5432 postgres:16-alpine

# Apply the schema
PGPASSWORD=devpassword psql -h localhost -U gforce -d gforce \
  -f internal/store/migrations/001_initial.sql

# Set required env vars
export GFORCE_DB_DSN="postgres://gforce:devpassword@localhost:5432/gforce?sslmode=disable"
export GFORCE_AUTH_JWT_SECRET="dev-secret-change-in-prod"

# Run with live reload
make dev

# Or build and run directly
make build
./gforce serve
```

### Useful make targets

| Target | Description |
|---|---|
| `make build` | Compile all Go binaries |
| `make test` | Run tests with race detector |
| `make lint` | Run golangci-lint |
| `make docker-build` | Build the container image |
| `make generate` | Regenerate CRD manifests |
| `make migrate` | Apply SQL migrations |
| `make dev` | Hot-reload server via air |

## Contributing

1. Fork the repository and create a feature branch.
2. Write tests alongside your code — `make test` must pass with `-race`.
3. Run `make lint` and fix any issues before opening a PR.
4. Follow the existing patterns: no global state, errors wrapped with context, structured logging everywhere.
5. For significant changes, open an issue first to discuss the design.

All contributions are welcome: bug fixes, documentation, new features, and operator improvements.

## License

Apache 2.0 — see [LICENSE](LICENSE) for details.
