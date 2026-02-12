--migrations/003_create_incident.up.sql
CREATE TABLE IF NOT EXISTS incidents (
    incident_id TEXT PRIMARY KEY,
    container_id TEXT,
    pod_name TEXT,
    namespace TEXT,
    risk_score INT,
    severity TEXT,
    signal_count INT,
    detected_at TIMESTAMP
);