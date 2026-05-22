# RESULT

## 结果摘要

- 新增 Python `LocalCodeInspector` 和 debug-only `Local Code Agent`。
- 决策层现在可以在 Gateway 证据不足且 `debug_local_code=true` 时，通过本地 registry 用 `service_name/repo_hint` 查 allowlist 仓库。
- 返回内容只包含 repo id、相对路径、命中词和行号，不返回源码片段、不返回本地绝对路径、不自动改代码。
- 无 mapping、未启用或无关键词时会收敛，不会乱读路径。

## 变更范围

- `apps/decision-engine/decision_engine/local_code.py`
- `apps/decision-engine/decision_engine/agent_team.py`
- `apps/decision-engine/decision_engine/models.py`
- `apps/decision-engine/tests/test_engine.py`
- `apps/decision-engine/README.md`
- `api/openapi/decision-engine.yaml`
- `docs/agent-framework-selection.md`
- `docs/decision-logging-and-limits.md`
- `README.md`
- `programs/P-2026-011-local-code-debug-inspection/*`

## 验证摘要

- Python decision-engine 单测：pass，12 tests。
- `make test`：pass。
- HTTP smoke：pass，命中 allowlist Java 文件，跳过生产配置。
- 敏感输出检查：pass，mock token 未进入输出 JSON。
- `python3.13 -m py_compile ...`：pass。
- `git diff --check`：pass。
- `python3.13 scripts/secret-scan.py --mode all`：pass。

## 验收覆盖

| 验收标准 | 结论 | Evidence ID |
| --- | --- | --- |
| Gateway 不下发/控制本地路径 | pass | EV-T1-001 |
| 本地 registry + allowlist 搜索可用 | pass | EV-T2-001 |
| debug 显式开启且 Gateway 证据不足才触发 | pass | EV-T3-001, EV-T4-001 |
| 敏感文件 deny，且不返回源码片段 | pass | EV-T4-001, EV-T5-003 |
| HTTP debug flow 可跑通 | pass | EV-T5-002 |

## Commit

- 本 Program 随本次提交交付；最终 hash 以 `git log` 为准。

## 残留风险

- 真实主链路尚未把 Gateway 工具结果不足状态二次传给 Python decision-engine。
- 当前是关键词级检索，不是 AST/call graph。
- 生产环境必须默认关闭本地代码检查。
