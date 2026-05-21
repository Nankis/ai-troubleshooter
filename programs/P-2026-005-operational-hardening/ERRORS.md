# ERRORS

## 已处理

- 本地环境 `go` 不在默认 PATH 时会导致 `make test` 失败；公开仓库已改为默认使用 PATH 中的 `go` 和 `gofmt`，开发者需先安装 Go 1.26+。

## 待观察

- 老数据库如果已经存在同一 `source + message_id` 或同一 `lark_message_id` 的重复数据，执行 `004_case_idempotency.sql` 前需要先清理重复行。
