# HANDOFF

当前目标：把本地代码辅助排查从“路径命中列表”升级成开发者可行动的代码定位报告。

已完成：

- 建立 Program。
- 明确 debug-only、allowlist、有界摘录和生产证据边界。
- `LocalCodeInspector` 已返回 primary symbol、line range、code excerpt、suspect reasons、follow-up checks。
- Agent Platform 本地代码回复已改为多行定位报告。
- 决策日志已压缩 local_code evidence，避免写入整段源码。
- 已用 MySQL + 真实本地 health-food 源码映射验证 `case_20260525_000049`。
- Web 已打开 `http://127.0.0.1:19091/web` 并点击 `case_20260525_000049`，页面包含方法名、代码行和行范围。

提交状态：

- Commit subject：`P-2026-043 make local code reports actionable`。
- 准备推送 `main`。

下一步：

- 无。后续若继续增强，可新开 Program 接 LSP/LSIF 或 UI 代码块折叠。

已运行命令：

- `PYTHONPATH=apps/decision-engine .venv/bin/python -m unittest apps/decision-engine/tests/test_engine.py`：18 tests OK。
- `PYTHONPATH=apps/agent-platform:apps/decision-engine .venv/bin/python -m unittest apps/agent-platform/tests/test_service_helpers.py`：3 tests OK。
- `make test`：pass。
- `make secret-scan`：pass。
- `git diff --check`：pass。
- API 验证生成 `case_20260525_000049`。
- Web 验证点击 `case_20260525_000049`，截图 `/tmp/p043_local_code_web_report.png`。

风险：

- 代码摘录会进入 Web 回复和平台消息表，必须限制在本地 debug-only、allowlist、短行、脱敏。
- 本地代码线索不是生产证据；真实根因仍要用 Gateway/DB/日志确认。
