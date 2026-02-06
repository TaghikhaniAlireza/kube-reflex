# Kube-Reflex

Kube-Reflex is a Kubernetes Detection & Response operator that listens to Falco security alerts
and automatically executes remediation actions inside the cluster.

## Core Features

- Receive Falco alerts via HTTP webhook
- Match alerts against ResponseRule custom resources
- Automated remediation:
  - Pod deletion
  - Network isolation using NetworkPolicy
- Kubernetes-native implementation using controller-runtime
- Production-grade, extensible architecture

## Project Status

Under active development

## Testing ON..

KubeBuilder: v4.11.1

Kubernetes:  1.35.0

Go OS/Arch:  linux/amd64

## License

TBD