ALTER TABLE `tb_troubleshoot_case`
  ADD COLUMN `case_title` VARCHAR(128) NULL COMMENT 'case display title' AFTER `case_no`,
  ADD KEY `idx_case_title` (`case_title`);
