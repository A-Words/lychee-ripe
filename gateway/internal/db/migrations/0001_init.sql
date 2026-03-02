CREATE TABLE IF NOT EXISTS batches (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  batch_id TEXT NOT NULL UNIQUE,
  trace_code TEXT NOT NULL UNIQUE,
  status TEXT NOT NULL CHECK(status IN ('pending_anchor', 'anchored', 'anchor_failed')),
  orchard_id TEXT NOT NULL,
  orchard_name TEXT NOT NULL,
  plot_id TEXT NOT NULL,
  plot_name TEXT,
  harvested_at TEXT NOT NULL,
  total INTEGER NOT NULL,
  green INTEGER NOT NULL,
  half INTEGER NOT NULL,
  red INTEGER NOT NULL,
  young INTEGER NOT NULL,
  unripe_count INTEGER NOT NULL,
  unripe_ratio REAL NOT NULL,
  unripe_handling TEXT NOT NULL DEFAULT 'sorted_out' CHECK(unripe_handling = 'sorted_out'),
  note TEXT,
  anchor_hash TEXT,
  confirm_unripe INTEGER NOT NULL DEFAULT 0,
  retry_count INTEGER NOT NULL DEFAULT 0,
  last_error TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_batches_status_created_at ON batches(status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_batches_created_at ON batches(created_at DESC);

CREATE TABLE IF NOT EXISTS anchor_proofs (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  batch_id TEXT NOT NULL UNIQUE REFERENCES batches(batch_id) ON DELETE CASCADE,
  tx_hash TEXT NOT NULL,
  block_number INTEGER NOT NULL,
  chain_id TEXT NOT NULL,
  contract_address TEXT NOT NULL,
  anchor_hash TEXT NOT NULL,
  anchored_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_anchor_proofs_tx_hash ON anchor_proofs(tx_hash);

CREATE TABLE IF NOT EXISTS reconcile_jobs (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  job_id TEXT NOT NULL UNIQUE,
  trigger_type TEXT NOT NULL CHECK(trigger_type IN ('manual', 'auto')),
  status TEXT NOT NULL CHECK(status IN ('accepted', 'running', 'completed', 'failed')),
  requested_count INTEGER NOT NULL,
  scheduled_count INTEGER NOT NULL,
  skipped_count INTEGER NOT NULL,
  error_message TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS reconcile_job_items (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  job_id TEXT NOT NULL REFERENCES reconcile_jobs(job_id) ON DELETE CASCADE,
  batch_id TEXT NOT NULL REFERENCES batches(batch_id) ON DELETE CASCADE,
  before_status TEXT NOT NULL,
  after_status TEXT NOT NULL,
  attempt_no INTEGER NOT NULL,
  error_message TEXT,
  created_at TEXT NOT NULL,
  UNIQUE(job_id, batch_id)
);

CREATE TABLE IF NOT EXISTS audit_logs (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  event_type TEXT NOT NULL,
  entity_type TEXT NOT NULL,
  entity_id TEXT NOT NULL,
  status TEXT,
  message TEXT,
  request_id TEXT,
  payload_json TEXT,
  created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_audit_logs_entity ON audit_logs(entity_type, entity_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_logs_event_time ON audit_logs(event_type, created_at DESC);
