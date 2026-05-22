ALTER TABLE `tb_troubleshoot_knowledge_item`
  ADD COLUMN `observed_case_count` INT NOT NULL DEFAULT 0 COMMENT 'observed case count' AFTER `knowledge_status`,
  ADD COLUMN `last_root_cause_category` VARCHAR(128) NULL COMMENT 'last root cause category' AFTER `observed_case_count`,
  ADD COLUMN `last_confirmed_reason` TEXT NULL COMMENT 'last confirmed reason' AFTER `last_root_cause_category`,
  ADD COLUMN `last_evolved_at` DATETIME NULL COMMENT 'last evolved time' AFTER `last_confirmed_reason`,
  ADD UNIQUE KEY `uk_knowledge_identity` (`issue_domain`, `issue_type`, `last_root_cause_category`),
  ADD KEY `idx_knowledge_status_evolved` (`knowledge_status`, `last_evolved_at`),
  ADD KEY `idx_last_root_cause_category` (`last_root_cause_category`);

CREATE TABLE `tb_troubleshoot_case_feedback` (
  `id` BIGINT AUTO_INCREMENT PRIMARY KEY COMMENT 'primary key',
  `case_id` BIGINT NOT NULL COMMENT 'case id',
  `rating` INT NULL COMMENT 'feedback rating',
  `ai_useful` TINYINT(1) NOT NULL DEFAULT 1 COMMENT 'whether ai result is useful',
  `wrong_conclusion` TINYINT(1) NOT NULL DEFAULT 0 COMMENT 'whether conclusion is wrong',
  `missing_key_information` TEXT NULL COMMENT 'missing key information',
  `missing_tools_json` TEXT NULL COMMENT 'missing tools json',
  `comment` TEXT NULL COMMENT 'feedback comment',
  `created_by` VARCHAR(128) NULL COMMENT 'feedback creator id',
  `status` TINYINT NOT NULL DEFAULT 1 COMMENT 'row status: 0 deleted, 1 active',
  `create_time` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'create time',
  `update_time` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT 'update time',
  KEY `idx_case_create_time` (`case_id`, `create_time`),
  KEY `idx_rating_create_time` (`rating`, `create_time`),
  KEY `idx_status_update_time` (`status`, `update_time`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='case feedback table';

CREATE TABLE `tb_troubleshoot_knowledge_evolution_run` (
  `id` BIGINT AUTO_INCREMENT PRIMARY KEY COMMENT 'primary key',
  `run_no` VARCHAR(64) NOT NULL COMMENT 'evolution run number',
  `case_id` BIGINT NOT NULL COMMENT 'case id',
  `knowledge_item_id` BIGINT NULL COMMENT 'knowledge item id',
  `trigger_type` VARCHAR(64) NOT NULL COMMENT 'trigger type',
  `input_snapshot_json` TEXT NOT NULL COMMENT 'input snapshot json',
  `output_summary` TEXT NULL COMMENT 'output summary',
  `decision` VARCHAR(64) NOT NULL COMMENT 'evolution decision',
  `created_knowledge_item` TINYINT(1) NOT NULL DEFAULT 0 COMMENT 'whether created knowledge item',
  `updated_knowledge_item` TINYINT(1) NOT NULL DEFAULT 0 COMMENT 'whether updated knowledge item',
  `error_message` TEXT NULL COMMENT 'error message',
  `status` TINYINT NOT NULL DEFAULT 1 COMMENT 'row status: 0 deleted, 1 active',
  `create_time` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'create time',
  `update_time` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT 'update time',
  UNIQUE KEY `uk_run_no` (`run_no`),
  KEY `idx_case_create_time` (`case_id`, `create_time`),
  KEY `idx_knowledge_create_time` (`knowledge_item_id`, `create_time`),
  KEY `idx_decision_create_time` (`decision`, `create_time`),
  KEY `idx_status_update_time` (`status`, `update_time`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='knowledge evolution run table';
