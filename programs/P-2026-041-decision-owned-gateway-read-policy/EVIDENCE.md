# EVIDENCE

## 变更证据

- `apps/decision-engine/decision_engine/agent_team.py`：Knowledge Agent 内部判断 realtime gate。
- `apps/agent-platform/agent_platform/service.py`：不再计算 `_requires_realtime`，只传递平台经验候选。
- `apps/decision-engine/tests/test_engine.py`：新增“请查真实数据 + 显式日期时高置信经验不能短路”的决策层单测。
- `README.md`、`docs/architecture-decisions.md`：图和文字改为 Decision Engine 生成 tool plan，Platform Tool Executor 只执行计划。

## 验证

| 命令 | 结果 |
| --- | --- |
| `make test` | PASS，Go 全量测试、Python decision/agent/root tests 通过 |
| `make secret-scan` | PASS |
| `git diff --check` | PASS |

## 结论

PASS。Gateway 读取策略已回到 Decision Engine；Agent Platform service 不再计算 realtime gate。
