CREATE UNIQUE INDEX uk_cases_source_message_id ON cases (source, message_id);

CREATE UNIQUE INDEX uk_case_messages_lark_message_id ON case_messages (lark_message_id);
