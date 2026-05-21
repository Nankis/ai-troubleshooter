CREATE TABLE ai_decision_logs (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  case_id BIGINT NOT NULL,
  investigation_id BIGINT NULL,
  agent_id VARCHAR(128) NOT NULL,
  decision_type VARCHAR(64) NOT NULL,
  reason TEXT NULL,
  input_snapshot_json JSON NULL,
  output_snapshot_json JSON NULL,
  selected_tools_json JSON NULL,
  status VARCHAR(32) NOT NULL,
  latency_ms INT NULL,
  error_message TEXT NULL,
  created_at DATETIME NOT NULL,
  KEY idx_case_created (case_id, created_at),
  KEY idx_investigation_created (investigation_id, created_at),
  KEY idx_decision_type_created (decision_type, created_at),
  KEY idx_status_created (status, created_at)
);
