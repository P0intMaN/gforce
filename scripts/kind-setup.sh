#!/bin/bash
set -euo pipefail

echo "==> Building GForce image..."
docker build -t gforce:latest .

echo "==> Loading image into kind cluster..."
kind load docker-image gforce:latest --name kind

echo "==> Adding Bitnami helm repo..."
helm repo add bitnami https://charts.bitnami.com/bitnami
helm repo update

echo "==> Updating Helm chart dependencies..."
helm dependency update ./charts/gforce

echo "==> Installing GForce..."
helm upgrade --install gforce ./charts/gforce \
  --namespace gforce-system \
  --create-namespace \
  --wait \
  --timeout 5m \
  --set server.jwtSecret="$(openssl rand -hex 32)"

echo "==> Waiting for pods..."
kubectl wait --for=condition=ready pod \
  -l app.kubernetes.io/name=gforce \
  -n gforce-system \
  --timeout=120s

echo ""
echo "==> GForce is running!"
kubectl get pods -n gforce-system

echo ""
echo "==> To access GForce, run:"
echo "  HTTP: kubectl port-forward -n gforce-system svc/gforce 8080:80"
echo "  SSH:  kubectl port-forward -n gforce-system svc/gforce 2222:2222"
echo ""
echo "  Then open http://localhost:8080"
