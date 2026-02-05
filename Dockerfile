FROM golang:1.23-bookworm

ENV DEBIAN_FRONTEND=noninteractive

# ---- Base packages ----
RUN apt-get update && apt-get install -y \
    ca-certificates \
    git \
    make \
    gcc \
    bash \
    bash-completion \
    && rm -rf /var/lib/apt/lists/*

# ---- Copy kubectl (offline) ----
COPY tools/kubectl /usr/local/bin/kubectl
RUN chmod +x /usr/local/bin/kubectl

# ---- Copy kubebuilder (offline) ----
COPY tools/kubebuilder_linux_amd64 /usr/local/bin/kubebuilder
RUN chmod +x /usr/local/bin/kubebuilder

# ---- Environment ----
ENV GO111MODULE=on
ENV KUBECONFIG=/root/.kube/config

WORKDIR /app

CMD ["sleep", "infinity"]