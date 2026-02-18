--migrations/001_init_falco_events.sql
CREATE TABLE IF NOT EXISTS falco_events (
  time TIMESTAMPTZ NOT NULL,
  rule TEXT NOT NULL,
  priority TEXT,
  source TEXT,
  hostname TEXT,
  tags TEXT[],
  output TEXT,
  output_fields JSONB
);

-- TimescaleDB optional: convert to hypertable if extension is installed
DO $$
BEGIN
  PERFORM create_hypertable('falco_events', 'time', if_not_exists => TRUE);
EXCEPTION
  WHEN undefined_function THEN
    NULL; -- TimescaleDB not installed, use standard table
END;
$$;

CREATE INDEX IF NOT EXISTS idx_falco_rule ON falco_events (rule);
CREATE INDEX IF NOT EXISTS idx_falco_priority ON falco_events (priority);
CREATE INDEX IF NOT EXISTS idx_falco_time ON falco_events (time DESC);