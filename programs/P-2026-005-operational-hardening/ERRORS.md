# ERRORS

## 已处理

- 本机 `go` 不在默认 PATH，改用 Makefile 固定的 `/Users/ginseng/sdk/go1.26.2/bin/go` 和 `gofmt`。

## 待观察

- 老数据库如果已经存在同一 `source + message_id` 或同一 `lark_message_id` 的重复数据，执行 `004_case_idempotency.sql` 前需要先清理重复行。
