# TASKS

## Task 1: [x] 建立 Program

- Evidence：`EV-T1-PROGRAM`

## Task 2: [x] 控制面 API 鉴权

- 文件：`internal/httpauth/*`、`cmd/dev-server/*`、`cmd/baseline-orchestrator/*`
- 验收：
  - 未带 token 返回 401。
  - token 正确才允许访问 case/knowledge/root-cause/feedback 控制面接口。

## Task 3: [x] 生产配置 fail-closed

- 文件：`internal/config/*`、`internal/gateway/*`、各 cmd 入口
- 验收：
  - prod gateway 未配置鉴权时启动失败。
  - prod lark-bot 未配置 verification token / allowed chats 时启动失败。

## Task 4: [x] Policy 边界修复

- 文件：`internal/policy/*`
- 验收：
  - agent 配置 allowed groups 时缺少 `chat_id` deny。

## Task 5: [x] 文档和验证

- 文件：`README.md`、`docs/*`、`configs/*`
- 验收：
  - `git diff --check` 通过。
  - `go vet ./...` 通过。
  - `make test` 通过。

## Task 6: [x] Tool audit 持久化

- 文件：`internal/audit/*`、`internal/storage/*`、`migrations/*`、`internal/gateway/*`
- 验收：
  - MySQL store 实现 audit sink。
  - Gateway 从 storage 使用持久化 audit sink。
  - DDL 与工具调用的 case ref 类型一致。
