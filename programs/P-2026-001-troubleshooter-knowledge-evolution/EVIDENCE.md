# EVIDENCE

## EV-T1-PROGRAM

- 状态：PASS
- 证据：新增 `PROGRAM.md`、`SCOPE.yml`、`TASKS.md`、`STATUS.yml`。
- 说明：本轮工作纳入项目内 Program，当前分支为 `codex/knowledge-evolution`，不直接 push main。

## EV-T2-TEST

- 状态：PASS
- 命令：`make test`
- 结果：通过。
- 覆盖：
  - `internal/evolution`：root cause 回填触发 knowledge item 和 evolution run。
  - `internal/lark`：Lark v2 payload 和 challenge。
  - `internal/connectors`：标准 HTTP readonly envelope。
  - 既有 caseflow、policy、masking、tool registry 测试。

## EV-T3-SMOKE

- 状态：PASS
- 命令：
  - `HTTP_PORT=19094 make dev`
  - `POST /lark/events`
  - `POST /cases/case_20260521_000001/root-cause`
  - `GET /knowledge?issue_domain=kline&issue_type=价格不一致`
- 结果：
  - case 进入 `NEED_HUMAN_CONFIRMATION` 后回填 root cause。
  - 回填响应包含 `"status":"DONE"`、`knowledge_item`、`evolution_run`。
  - `/knowledge` 返回 `external_source_delay` 且 `observed_case_count=1`。

## EV-T4-FINAL

- 状态：AUX_PASS
- 命令：
  - `git diff --check`
  - `make test`
- 结果：
  - `git diff --check` 通过。
  - `make test` 通过。
- SKIP：
  - 未执行真实 MySQL migration，因为本机没有本项目测试 MySQL DSN；部署前需在测试库执行 `001_initial.sql` 和 `002_knowledge_evolution.sql`。
