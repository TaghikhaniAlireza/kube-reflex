# Kube-Reflex: Kubernetes Detection & Response Engine

## Introduction

Kube-Reflex is a stateful security response engine for Kubernetes that consumes Falco runtime security alerts and correlates them into actionable incidents. Unlike stateless alert handlers, Kube-Reflex maintains context across events: it tracks attack chains (MITRE ATT&CK tactics in sequence), detects velocity anomalies (high-frequency alert bursts), and aggregates signals before triggering remediation. The result is fewer false positives and higher-confidence response actions.

## Core Concepts & Workflow

Events flow through the following pipeline:

```
Falco Alert
    |
    v
HTTP Webhook (POST /api/v1/alerts)
    |
    v
Scoring (priority -> numeric score)
    |
    v
Taxonomy Mapper (Falco tags -> MITRE tactic ID)
    |
    +---> Velocity Engine (Redis) ----> Decision Engine (aggregation)
    |                                         |
    +---> FSM Engine (Redis) -----------------+
    |         (attack chain detection)         |
    |                                         v
    |                                    Judge (risk calc, K8s enrichment)
    |                                         |
    |                                         v
    +---------------------------------> Action Sinks (Postgres, Stdout)
```

1. **Ingest:** Falco sends JSON alerts to the webhook. Each event is validated and scored by priority.
2. **Correlation:** The taxonomy mapper maps Falco rule tags to MITRE tactic IDs. Events feed two parallel paths:
   - **Velocity:** Detects high-frequency alert bursts per container within configurable time windows.
   - **FSM:** Tracks state per container and chain; completes when a full attack sequence is observed.
3. **Decision:** The decision engine aggregates signals over a time window, enriches with Kubernetes metadata (pod, namespace), and computes a risk score.
4. **Action:** Incidents above the threshold are dispatched to sinks (PostgreSQL, stdout).

## Key Features

- **Stateful Analysis:** Redis-backed FSM and velocity tracking maintain context across events.
- **Attack Chain Detection:** Configurable MITRE ATT&CK tactic sequences in `chains.yml`; FSM detects multi-step attack patterns.
- **Velocity Detection:** Identifies alert flooding and pressure anomalies within sliding windows.
- **Real-time Processing:** Event-driven pipeline with buffered channels; non-blocking webhook.
- **Configurable Rules:** YAML-based chains and taxonomy; no code changes for new detection logic.
- **Kubernetes Integration:** In-cluster pod/namespace resolution; RBAC for read-only API access.
- **Structured Logging:** slog-based JSON logging with contextual fields for production observability.

## Installation & Deployment

For detailed deployment instructions, including Kubernetes manifests, Helm configuration for Falco, and secrets setup, see **[DEPLOY.md](DEPLOY.md)**.

## Configuration

All configuration is via environment variables:

| Variable | Purpose | Default |
|----------|---------|---------|
| `DATABASE_URL` | PostgreSQL connection string for incident storage and migrations | (required) |
| `REDIS_ADDR` | Redis server address for FSM state and velocity tracking | `localhost:6379` |
| `REDIS_PASSWORD` | Redis authentication password (optional) | (none) |
| `LISTEN_ADDR` | HTTP listen address for webhook and health endpoint | `:8080` |

Rule configuration is file-based and baked into the image by default:

| File | Purpose |
|------|---------|
| `internal/correlator/rules/chains.yml` | Attack chain definitions (MITRE tactic sequences, severity, max duration) |
| `internal/correlator/taxonomy/behaviors.yml` | MITRE ID to tactic mapping for Falco tag resolution |

These can be overridden at runtime via ConfigMap mounts (see `k8s/deployment.yaml` comments).

## Building from Source

```bash
go build -o brain ./cmd/brain
```

For a minimal production binary:

```bash
CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o brain ./cmd/brain
```

## Project Structure

```
kube-reflex/
├── cmd/
│   ├── brain/           # Main application entry point
│   └── test_k8s_redis/  # Integration test utility
├── internal/
│   ├── action/          # Action sinks (Postgres, stdout)
│   ├── correlator/
│   │   ├── fsm/         # State machine for attack chain detection (Redis store)
│   │   ├── rules/      # Chain loader and registry
│   │   ├── taxonomy/   # MITRE tag mapper (behaviors.yml)
│   │   └── velocity/   # Velocity detector and engine
│   ├── decision/       # Aggregation, Judge, risk scoring
│   ├── domain/         # Incident domain model
│   ├── db/             # PostgreSQL repositories and migrations
│   ├── falco/          # Webhook handler, event decoding, validation
│   ├── k8s/            # Kubernetes client and pod resolver
│   ├── logger/         # Structured logging interface (slog)
│   ├── model/          # Alert, chain, behavior models
│   ├── parser/         # Identity extraction from Falco output fields
│   ├── redis/          # Redis client and container state repository
│   └── scoring/        # Priority-to-score mapping
├── migrations/         # PostgreSQL schema migrations
├── k8s/               # Deployment, Service, RBAC, ConfigMap
├── deploy/            # Falco Helm values for HTTP output
└── scripts/           # Falco simulation and utilities
```

## License

TBD

## Community & Support

This project is developed as **Open Source** with the goal of enhancing security in the Kubernetes ecosystem. We warmly welcome any contributions, suggestions, or bug reports.

If you have an idea to improve Kube-Reflex or would like to contribute to its development:
1.  Find an issue you'd like to work on (or open a new one).
2.  **Fork** this repository.
3.  Make your changes and submit a **Pull Request**.

For direct communication, technical discussions, or to talk about the project roadmap, you can connect with me on LinkedIn:

<div align="center">

[![LinkedIn](https://img.shields.io/badge/LinkedIn-Connect-blue?style=for-the-badge&logo=linkedin)](https://www.linkedin.com/in/alireza-taghikhani/)
[![GitHub](https://img.shields.io/badge/GitHub-Follow-black?style=for-the-badge&logo=github)](https://github.com/TaghikhaniAlireza/)

</div>
