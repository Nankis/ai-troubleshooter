CREATE UNIQUE INDEX `uk_case_source_message_id` ON `tb_troubleshoot_case` (`source`, `message_id`);

CREATE UNIQUE INDEX `uk_case_message_platform_message_id` ON `tb_troubleshoot_case_message` (`platform_message_id`);
