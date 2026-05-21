# TASKS

## Task 1: [x] 建立 Program

- Evidence：`EV-T1-PROGRAM`

## Task 2: [x] Gateway HTTP 鉴权与身份绑定

- 文件：`internal/gateway/*`、`internal/config/*`
- 验收：
  - 未认证返回 401。
  - agent_id mismatch 返回 403。

## Task 3: [x] 网关级限流

- 文件：`internal/ratelimit/*`、`internal/gateway/*`
- 验收：
  - 超过配置 QPS 返回 429。

## Task 4: [x] 文档和验证

- 文件：`docs/*`、`README.md`、`programs/...`
- 验收：
  - `git diff --check` 通过。
  - `make test` 通过。
