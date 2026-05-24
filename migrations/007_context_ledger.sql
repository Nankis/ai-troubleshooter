CREATE TABLE IF NOT EXISTS `tb_troubleshoot_context_ledger` (
  `id` BIGINT AUTO_INCREMENT PRIMARY KEY COMMENT 'primary key',
  `case_id` BIGINT NOT NULL COMMENT 'case id',
  `ledger_type` VARCHAR(64) NOT NULL COMMENT 'ledger type: case_state/tool_evidence/agent_report/final_summary',
  `ledger_key` VARCHAR(128) NOT NULL DEFAULT '' COMMENT 'ledger key or sub step',
  `summary` TEXT NULL COMMENT 'compact summary allowed into agent context',
  `evidence_refs_json` TEXT NULL COMMENT 'evidence references json, such as tool_call_id or downstream query id',
  `payload_json` TEXT NULL COMMENT 'redacted compact payload json; raw tool data is not stored here',
  `source_agent` VARCHAR(128) NULL COMMENT 'source agent or component',
  `status` TINYINT NOT NULL DEFAULT 1 COMMENT 'row status: 0 deleted, 1 active',
  `create_time` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'create time',
  `update_time` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT 'update time',
  KEY `idx_case_type_create_time` (`case_id`, `ledger_type`, `create_time`),
  KEY `idx_case_key` (`case_id`, `ledger_key`),
  KEY `idx_status_update_time` (`status`, `update_time`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='troubleshooting context ledger table';
