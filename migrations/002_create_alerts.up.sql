--migrations/002_create_alerts.sql
CREATE TABLE IF NOT EXISTS alerts (
    alert_id UUID PRIMARY KEY,
    container_id TEXT NOT NULL,
    chain_id TEXT NOT NULL,
    severity TEXT NOT NULL,
    type TEXT NOT NULL,
    occurred_at TIMESTAMP WITH TIME ZONE NOT NULL,
    payload JSONB NOT NULL, -- Stores the full Alert struct for flexibility
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_alerts_container_id ON alerts(container_id);
CREATE INDEX idx_alerts_severity ON alerts(severity);
CREATE INDEX idx_alerts_occurred_at ON alerts(occurred_at DESC);