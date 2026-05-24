# HANDOFF

当前目标：澄清 Decision Engine 运行形态，防止把“不单独启动”误解为“不使用”。

已完成：

- 建立 Program。
- 确认代码主路径：Agent Platform 进程内持有 `DecisionEngine` 并调用 `plan()`。
- 新增 Web Chat 回归测试：`test_web_chat_must_call_decision_engine_plan`。
- 文档已澄清：正常 Web Chat 不单独启动外部 Decision Engine 服务，但必须进程内调用 `DecisionEngine.plan()`。

下一步：

- 提交并推送 `main`。

已运行命令：

- `PYTHONPATH=apps/agent-platform:apps/decision-engine .venv/bin/python -m unittest apps/agent-platform/tests/test_agent_platform_fastapi.py`：16 tests OK。
- `PYTHONPATH=apps/decision-engine .venv/bin/python -m unittest apps/decision-engine/tests/test_engine.py`：18 tests OK。
- `make test`：pass。
- `make secret-scan`：pass。
- `git diff --check`：pass。

提交状态：

- Commit subject：`P-2026-044 clarify decision engine runtime`。
- 准备推送 `main`。

下一步：

- 无。后续如果要把 Decision Engine 拆成独立服务，需要另开 Program 设计部署、鉴权、观测和调用链。

风险：

- 本轮只澄清运行形态，不改变部署拓扑。
