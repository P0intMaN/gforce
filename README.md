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

## Deploy on Kubernetes (kind)

**Prerequisites:** `kind`, `helm`, `kubectl`, `docker`

```bash
# Create the kind cluster (1 control-plane + 2 workers)
kind create cluster --config kind-config.yaml

# One-command build + deploy
make kind-setup

# Access the platform
kubectl port-forward -n gforce-system svc/gforce 8080:80 &
kubectl port-forward -n gforce-system svc/gforce 2222:2222 &

# Open http://localhost:8080
curl http://localhost:8080/healthz

# Git over HTTP
git clone http://localhost:8080/owner/repo.git

# Git over SSH
git clone ssh://git@localhost:2222/owner/repo.git

# Inspect CRDs
kubectl get repositories -n gforce-system
kubectl get gforceusers -n gforce-system

# Tear down
make kind-teardown
```

## Deploy on any Kubernetes cluster

```bash
helm repo add bitnami https://charts.bitnami.com/bitnami
helm dependency update ./charts/gforce

helm install gforce ./charts/gforce \
  --namespace gforce-system \
  --create-namespace \
  --set server.baseURL=https://git.mycompany.com \
  --set ingress.enabled=true \
  --set ingress.host=git.mycompany.com \
  --set server.jwtSecret="$(openssl rand -hex 32)"
```

## Quickstart (local dev)

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
| `make docker-build` | Build the container image (`gforce:latest`) |
| `make kind-load` | Load image into kind cluster |
| `make helm-lint` | Lint the Helm chart |
| `make helm-template` | Render Helm templates to stdout |
| `make helm-install` | Install/upgrade via Helm |
| `make helm-uninstall` | Uninstall from cluster |
| `make kind-setup` | Build, load, and install in one step |
| `make kind-teardown` | Remove GForce from kind |
| `make port-forward` | Forward HTTP (8080) and SSH (2222) ports |
| `make generate` | Regenerate CRD manifests |
| `make migrate` | Run DB migrations locally |
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
