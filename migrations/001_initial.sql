CREATE TABLE cases (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  case_no VARCHAR(64) NOT NULL UNIQUE,
  source VARCHAR(32) NOT NULL,
  chat_id VARCHAR(128) NULL,
  thread_id VARCHAR(128) NULL,
  message_id VARCHAR(128) NULL,
  reporter_user_id VARCHAR(128) NULL,
  original_text TEXT NULL,
  ocr_text TEXT NULL,
  issue_domain VARCHAR(64) NULL,
  issue_type VARCHAR(64) NULL,
  status VARCHAR(64) NOT NULL,
  priority VARCHAR(32) NOT NULL DEFAULT 'normal',
  timezone VARCHAR(64) NOT NULL DEFAULT 'Asia/Shanghai',
  occurred_at DATETIME NULL,
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL,
  closed_at DATETIME NULL,
  version BIGINT NOT NULL DEFAULT 0,
  KEY idx_status_updated (status, updated_at),
  KEY idx_domain_type (issue_domain, issue_type)
);

CREATE TABLE case_entities (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  case_id BIGINT NOT NULL,
  entity_type VARCHAR(64) NOT NULL,
  entity_value VARCHAR(255) NOT NULL,
  source VARCHAR(32) NOT NULL,
  confidence DECIMAL(5,4) NULL,
  created_at DATETIME NOT NULL,
  KEY idx_case_entity (case_id, entity_type),
  KEY idx_entity_value (entity_type, entity_value)
);

CREATE TABLE case_messages (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  case_id BIGINT NOT NULL,
  role VARCHAR(32) NOT NULL,
  lark_message_id VARCHAR(128) NULL,
  content TEXT NOT NULL,
  content_type VARCHAR(32) NOT NULL DEFAULT 'text',
  created_at DATETIME NOT NULL,
  KEY idx_case_created (case_id, created_at)
);

CREATE TABLE investigations (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  investigation_no VARCHAR(64) NOT NULL UNIQUE,
  case_id BIGINT NOT NULL,
  agent_id VARCHAR(128) NOT NULL,
  agent_version VARCHAR(64) NULL,
  model_provider VARCHAR(64) NULL,
  model_name VARCHAR(128) NULL,
  status VARCHAR(64) NOT NULL,
  initial_hypothesis TEXT NULL,
  final_summary TEXT NULL,
  confidence DECIMAL(5,4) NULL,
  started_at DATETIME NOT NULL,
  finished_at DATETIME NULL,
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL,
  KEY idx_case (case_id),
  KEY idx_status_updated (status, updated_at)
);

CREATE TABLE tool_call_audits (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  tool_call_id VARCHAR(64) NOT NULL UNIQUE,
  case_ref VARCHAR(64) NOT NULL,
  investigation_ref VARCHAR(64) NULL,
  agent_id VARCHAR(128) NOT NULL,
  lark_user_id VARCHAR(128) NULL,
  tool_name VARCHAR(128) NOT NULL,
  required_scope VARCHAR(128) NULL,
  arguments_summary TEXT NULL,
  policy_decision VARCHAR(32) NOT NULL,
  deny_reason VARCHAR(255) NULL,
  query_id VARCHAR(128) NULL,
  result_count INT NULL,
  latency_ms INT NULL,
  error_message TEXT NULL,
  created_at DATETIME NOT NULL,
  KEY idx_case_created (case_ref, created_at),
  KEY idx_tool_created (tool_name, created_at),
  KEY idx_agent_created (agent_id, created_at)
);

CREATE TABLE root_causes (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  case_id BIGINT NOT NULL UNIQUE,
  ai_predicted_reason TEXT NULL,
  human_confirmed_reason TEXT NOT NULL,
  root_cause_category VARCHAR(128) NOT NULL,
  owner_service VARCHAR(128) NULL,
  owner_team VARCHAR(128) NULL,
  is_cache_issue BOOLEAN NOT NULL DEFAULT FALSE,
  is_data_sync_issue BOOLEAN NOT NULL DEFAULT FALSE,
  is_external_source_issue BOOLEAN NOT NULL DEFAULT FALSE,
  is_frontend_display_issue BOOLEAN NOT NULL DEFAULT FALSE,
  is_user_misunderstanding BOOLEAN NOT NULL DEFAULT FALSE,
  fix_action TEXT NULL,
  prevention_action TEXT NULL,
  confirmed_by VARCHAR(128) NULL,
  confirmed_at DATETIME NOT NULL,
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL
);

CREATE TABLE knowledge_items (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  title VARCHAR(255) NOT NULL,
  issue_domain VARCHAR(64) NOT NULL,
  issue_type VARCHAR(64) NULL,
  typical_description TEXT NULL,
  typical_ocr_features TEXT NULL,
  required_fields_json JSON NULL,
  recommended_steps_json JSON NULL,
  common_causes_json JSON NULL,
  useful_tools_json JSON NULL,
  success_case_ids_json JSON NULL,
  failure_case_ids_json JSON NULL,
  confidence DECIMAL(5,4) NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'active',
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL,
  KEY idx_domain_type (issue_domain, issue_type)
);

CREATE TABLE agent_registry (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  agent_id VARCHAR(128) NOT NULL UNIQUE,
  name VARCHAR(128) NOT NULL,
  client_id VARCHAR(128) NOT NULL UNIQUE,
  client_secret_hash VARCHAR(255) NOT NULL,
  allowed_scopes_json JSON NOT NULL,
  allowed_tools_json JSON NOT NULL,
  allowed_lark_groups_json JSON NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'enabled',
  rate_limit_qps INT NOT NULL DEFAULT 5,
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL
);

CREATE TABLE tool_registry (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  tool_name VARCHAR(128) NOT NULL UNIQUE,
  description TEXT NOT NULL,
  input_schema_json JSON NOT NULL,
  output_schema_json JSON NULL,
  required_scope VARCHAR(128) NOT NULL,
  backend_handler VARCHAR(128) NOT NULL,
  max_time_range_minutes INT NULL,
  max_limit INT NULL,
  sensitivity_level VARCHAR(32) NOT NULL DEFAULT 'normal',
  status VARCHAR(32) NOT NULL DEFAULT 'enabled',
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL
);
