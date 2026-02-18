# Kube-Reflex Deployment Guide

This guide describes how to deploy Kube-Reflex with Falco using standard Kubernetes and Helm practices.

## Prerequisites

- Kubernetes cluster (1.24+)
- Helm 3.x
- PostgreSQL (for incident storage)
- Redis (for FSM state and velocity tracking)

## 1. Deploy Kube-Reflex

### Create Secrets and Config

```bash
# Create the namespace (if not using default)
kubectl create namespace default

# Create secret for database URL
kubectl create secret generic kube-reflex-secrets \
  --from-literal=database-url="postgres://user:pass@postgres:5432/kube_reflex?sslmode=disable" \
  -n default

# Optional: Add Redis password to secret if using authenticated Redis
# kubectl patch secret kube-reflex-secrets -n default --type='json' -p='[{"op": "add", "path": "/redis-password", "value": "<base64-encoded-password>"}]'

# Apply ConfigMap for Redis address
kubectl apply -f k8s/configmap.yaml -n default

# Apply RBAC
kubectl apply -f k8s/rbac.yaml -n default
```

### Deploy Application

```bash
# Build and push the image (or use your registry)
docker build -t kube-reflex:latest .
# docker push <your-registry>/kube-reflex:latest

# Deploy Kube-Reflex
kubectl apply -f k8s/deployment.yaml -n default
kubectl apply -f k8s/service.yaml -n default
```

### Verify Kube-Reflex

```bash
kubectl get pods -l app=kube-reflex -n default
kubectl get svc kube-reflex -n default
```

The service `kube-reflex` must be available on port 8080 for Falco to send alerts.

## 2. Install Falco with Helm

Falco is installed using the official Falco Helm chart, configured to send alerts to Kube-Reflex.

### Add Helm Repository

```bash
helm repo add falcosecurity https://falcosecurity.github.io/charts
helm repo update
```

### Install Falco with Custom Values

```bash
helm install falco falcosecurity/falco -f deploy/falco-values.yaml -n default
```

Or, if installing Falco in a different namespace (e.g. `falco`):

```bash
# Edit deploy/falco-values.yaml and change the URL to use the full DNS:
# http_output.url=http://kube-reflex.default.svc.cluster.local:8080/api/v1/alerts
# (Kube-Reflex in default namespace; Falco can be in any namespace)

helm install falco falcosecurity/falco -f deploy/falco-values.yaml -n falco
```

### Upgrade Existing Falco

```bash
helm upgrade falco falcosecurity/falco -f deploy/falco-values.yaml -n default
```

### Verify Falco

```bash
kubectl get pods -l app.kubernetes.io/name=falco -n default
kubectl logs -l app.kubernetes.io/name=falco -n default -f
```

## 3. Configuration Summary

| Component    | Setting              | Value                                                                 |
|-------------|----------------------|-----------------------------------------------------------------------|
| Falco       | HTTP Output          | Enabled                                                               |
| Falco       | Webhook URL          | `http://kube-reflex.default.svc.cluster.local:8080/api/v1/alerts`    |
| Falco       | JSON Output          | `true`                                                                |
| Falco       | Log Level            | `info`                                                                |
| Kube-Reflex | Service Name         | `kube-reflex`                                                         |
| Kube-Reflex | Port                 | `8080`                                                                |
| Kube-Reflex | Webhook Path         | `/api/v1/alerts`                                                      |

## 4. Different Namespaces

If Kube-Reflex is deployed in a namespace other than `default`, update `deploy/falco-values.yaml`:

```yaml
# Change the URL to match your namespace, e.g.:
# http_output.url=http://kube-reflex.<your-namespace>.svc.cluster.local:8080/api/v1/alerts
```

Apply the same namespace to all `kubectl` and `helm` commands.
