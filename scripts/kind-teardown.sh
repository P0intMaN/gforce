#!/bin/bash
set -euo pipefail

echo "==> Uninstalling GForce..."
helm uninstall gforce -n gforce-system || true

echo "==> Deleting namespace..."
kubectl delete namespace gforce-system --ignore-not-found

echo "==> Done."
