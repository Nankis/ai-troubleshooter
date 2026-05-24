# Handoff

## 当前目标

P-2026-036：升级 Agent 排查链路，避免单主 Agent 吞入全量工具结果导致上下文膨胀。通过 Context Ledger、压缩 observation、证据引用和最终答案 verifier，把事实证据与 LLM 上下文分层。

## 已完成

- 新增 `tb_troubleshoot_context_ledger` migration。
- 新增 Python `context_ledger.py`，压缩工具观测并生成证据引用。
- Agent Platform 写入 case state、gateway tools、knowledge retrieval、agent report、tool evidence、final summary ledger。
- `_summarize()` 只接收压缩 observation，不再传入 Gateway 原始 `data`。
- 新增 `verifier_final_answer` 决策日志。
- Decision Engine `DecisionRequest` 支持 `context_ledger`，Supervisor/Verifier 记录短上下文策略。
- README、架构文档、决策日志文档已更新。
- 已跑 Agent Platform 和 Decision Engine 目标单测。
- 修复 MySQL `Decimal` 等数据库标量类型写 ledger JSON 的序列化问题，并补测试。
- 已完成全量测试、secret scan、diff check。
- 已完成本地 MySQL migration 和 API 写入查询验证，代表 case：`case_20260524_000022`。

## 证据路径

- `programs/P-2026-036-context-ledger-agent-isolation/EVIDENCE.md`

## 已运行命令

- `PYTHONPATH=apps/agent-platform:apps/decision-engine .venv/bin/python -m unittest apps/agent-platform/tests/test_agent_platform_fastapi.py`
- `PYTHONPATH=apps/agent-platform:apps/decision-engine .venv/bin/python -m unittest apps/decision-engine/tests/test_engine.py`
- `make test`
- `make secret-scan`
- `git diff --check`
- `MYSQL_PASSWORD=<local> MYSQL_DATABASE=ai_troubleshooter make migrate-mysql`
- 本地启动 Gateway `:18080` 和 Agent Platform `:19091`，通过 `/api/v1/chat` 生成 `case_20260524_000022`。
- MySQL 查询 `tb_troubleshoot_context_ledger`，确认 ledger 类型和原始 `"data"` key 计数。
- 本地 Gateway `:18080` 和 Agent Platform `:19091` 已停止。

## 工作树

- 已提交 `P-2026-036 add context ledger isolation`，仍需 push。

## 下一步

1. push main。

## 风险/阻塞

- 本轮没有接真实 LLM/Vision 或真实 health-food 生产 readonly adapter；只能声明 Context Ledger 和上下文隔离链路已在本地 MySQL + mock Gateway 下验证。
