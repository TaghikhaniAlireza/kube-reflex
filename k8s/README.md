# Kube-Reflex Kubernetes Deployment

## Prerequisites

- Kubernetes cluster with Redis and PostgreSQL available (or deploy them separately).
- Falco installed (e.g. via Helm) if you want to send alerts to Kube-Reflex.

## 1. Create the database secret

Create a Secret with your Postgres connection string (replace with your values):

```bash
kubectl create secret generic kube-reflex-secrets \
  --from-literal=database-url='postgres://user:password@postgres:5432/kube_reflex?sslmode=disable'
```

## 2. Deploy Kube-Reflex

Apply ConfigMap, RBAC, Deployment, and Service (default namespace):

```bash
kubectl apply -f k8s/configmap.yaml
kubectl apply -f k8s/rbac.yaml
kubectl apply -f k8s/deployment.yaml
kubectl apply -f k8s/service.yaml
```

If you use another namespace, create it first and add `-n <namespace>` to each command, and update `k8s/rbac.yaml` RoleBinding `subjects[].namespace` to match.

## 3. Build and use the image

Build the brain image and load into your cluster (or push to a registry):

```bash
docker build -t kube-reflex:latest .
# For kind: kind load docker-image kube-reflex:latest
# For minikube: minikube image load kube-reflex:latest
```

Then ensure the Deployment's `image` (e.g. `kube-reflex:latest`) matches.

## 4. Point Falco to Kube-Reflex

Use the provided Helm values override so Falco sends alerts to this service:

```bash
helm repo add falcosecurity https://falcosecurity.github.io/charts
helm repo update
helm upgrade --install falco falcosecurity/falco -f falco-values.yaml
```

The override configures Falco to POST each alert to `http://kube-reflex-service:8080/api/v1/alerts`. If Falco is in a different namespace, edit `falco-values.yaml` and use the full DNS name: `http://kube-reflex-service.<namespace>.svc.cluster.local:8080/api/v1/alerts`.

## Optional: ConfigMaps for chains/behaviors

To override `chains.yml` or `behaviors.yml` without rebuilding the image, create ConfigMaps and uncomment the volume/volumeMount sections in `deployment.yaml`.
