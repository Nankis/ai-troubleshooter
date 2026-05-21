# TASKS

## Task 1: [x] 建立 Program

- Evidence：`EV-T1-PROGRAM`

## Task 2: [x] 决策日志模型和持久化

- 文件：`internal/caseflow/*`、`internal/storage/*`、`migrations/*`
- 验收：
  - memory/mysql store 均支持决策日志写入与查询。

## Task 3: [x] Orchestrator 记录决策过程

- 文件：`internal/orchestrator/*`
- 验收：
  - 分类、实体抽取、字段检查、工具计划、工具调用、总结都写日志。

## Task 4: [x] 超时和查询限制

- 文件：`internal/orchestrator/*`、`internal/config/*`
- 验收：
  - case 级 timeout。
  - tool call 上限。
  - tool failure 上限。
  - 失败后 case/investigation 收敛。

## Task 5: [x] 文档和验证

- 文件：`README.md`、`docs/*`、`configs/*`、`programs/...`
- 验收：
  - `git diff --check` 通过。
  - `go vet ./...` 通过。
  - `make test` 通过。
  - `go test -race ./...` 通过。
