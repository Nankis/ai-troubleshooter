# TASKS

## Task 1: [x] 建立 Program

- Evidence：`EV-T1-PROGRAM`

## Task 2: [x] Lark 事件幂等

- 文件：`internal/lark/*`、`internal/caseflow/*`、`internal/storage/mysql/*`
- 验收：
  - 重复 `message_id` 返回已有 case。
  - 不重复 publish queue event。

## Task 3: [x] Decision runner 重复处理保护

- 文件：`internal/decisionbaseline/*`、`internal/caseflow/state.go`
- 验收：
  - 仅入口状态允许开始处理。
  - 重复处理写入 `process_skipped`。

## Task 4: [x] 决策日志快照脱敏

- 文件：`internal/decisionbaseline/*`
- 验收：
  - 决策日志快照不出现手机号、token、api key 明文。

## Task 5: [x] 文档和验证

- 文件：`README.md`、`docs/*`、`migrations/*`
- 验收：
  - `git diff --check` 通过。
  - `go vet ./...` 通过。
  - `make test` 通过。
  - `go test -race ./...` 通过。
