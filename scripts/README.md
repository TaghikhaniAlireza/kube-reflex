# Scripts

## simulate_falco.go – send dummy Falco alerts to Kube-Reflex

Sends HTTP POST requests to the webhook so you can verify FSM (attack chain) and velocity (volume) logic without a real Falco instance.

### Prerequisites

- Brain service running and listening on `http://localhost:8080` (or set `LISTEN_ADDR` and use that URL).
- Redis and Postgres available if the brain uses them (otherwise events may be accepted but processing can fail).

### How to run

```bash
# From repo root, with brain already running in another terminal
go run scripts/simulate_falco.go
```

Optional: override the webhook URL:

```bash
go run scripts/simulate_falco.go http://localhost:8080/api/v1/alerts
go run scripts/simulate_falco.go http://kube-reflex-service:8080/api/v1/alerts   # in-cluster
```

### What the script sends

1. **FSM (attack chain)** – 3 events in order, same container:
   - **Recon** – "Network reconnaissance tool executed in container" (T1595 → TA0043).
   - **Initial Access** – "Suspicious download inside container" (T1568 → TA0011).
   - **Terminal Shell** – "Terminal shell in container" (T1059 → TA0002).

   This completes the `remote_exploit_sample` chain (TA0043 → TA0011 → TA0002).

2. **Velocity** – 5 rapid **Sensitive file opened for reading by non-trusted program** (T1555) alerts, same container, to trigger high-frequency detection.

### Expected brain logs

- **Webhook:** Each POST should return `202 Accepted`; script prints `✓ Rule (Priority) -> 202 Accepted` per event.
- **FSM:**
  - `🟢 FSM START | Chain=remote_exploit_sample | Tactic=TA0043`
  - `🔼 FSM PROMOTE | Chain=remote_exploit_sample | Step=1 | Tactic=TA0011`
  - `🚨 CHAIN COMPLETED | Chain=remote_exploit_sample | Severity=CRITICAL`
- **Velocity (if threshold is met):**  
  `[velocity] SCORE container=sim-test-container-001 rule=VEL_WARN_SPAM score=...`
- **Decision:** After the aggregation window, you may see action engine output (e.g. stdout sink) for the incident with severity CRITICAL or HIGH.

If you see `[brain] skip event without container_id` or `[brain] mapper skip rule=...`, the payload is missing `container.id` in `output_fields` or a valid MITRE tag in `tags`; the script includes both.
