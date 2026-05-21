ALTER TABLE knowledge_items
  ADD COLUMN observed_case_count INT NOT NULL DEFAULT 0 AFTER status,
  ADD COLUMN last_root_cause_category VARCHAR(128) NULL AFTER observed_case_count,
  ADD COLUMN last_confirmed_reason TEXT NULL AFTER last_root_cause_category,
  ADD COLUMN last_evolved_at DATETIME NULL AFTER last_confirmed_reason,
  ADD UNIQUE KEY uk_knowledge_identity (issue_domain, issue_type, last_root_cause_category),
  ADD KEY idx_status_evolved (status, last_evolved_at),
  ADD KEY idx_root_cause_category (last_root_cause_category);

CREATE TABLE case_feedbacks (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  case_id BIGINT NOT NULL,
  rating INT NULL,
  ai_useful BOOLEAN NOT NULL DEFAULT TRUE,
  wrong_conclusion BOOLEAN NOT NULL DEFAULT FALSE,
  missing_key_information TEXT NULL,
  missing_tools_json JSON NULL,
  comment TEXT NULL,
  created_by VARCHAR(128) NULL,
  created_at DATETIME NOT NULL,
  KEY idx_case_created (case_id, created_at),
  KEY idx_rating_created (rating, created_at)
);

CREATE TABLE knowledge_evolution_runs (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  run_no VARCHAR(64) NOT NULL UNIQUE,
  case_id BIGINT NOT NULL,
  knowledge_item_id BIGINT NULL,
  trigger_type VARCHAR(64) NOT NULL,
  input_snapshot_json JSON NOT NULL,
  output_summary TEXT NULL,
  decision VARCHAR(64) NOT NULL,
  created_knowledge_item BOOLEAN NOT NULL DEFAULT FALSE,
  updated_knowledge_item BOOLEAN NOT NULL DEFAULT FALSE,
  error_message TEXT NULL,
  created_at DATETIME NOT NULL,
  KEY idx_case_created (case_id, created_at),
  KEY idx_knowledge_created (knowledge_item_id, created_at),
  KEY idx_decision_created (decision, created_at)
);
