# HANDOFF

当前目标：增强 Web 左侧问题列表状态，支持排查中 spinner 和 AI 结果待查看提示。

已完成：

- 建立 Program。
- 明确“待查看”为前端本机读回执，不改变后端 case 状态机。
- 修改 `web/static/index.html`：
  - `INVESTIGATING` / `READY_TO_INVESTIGATE` / `WAITING_TOOL_RESULT` 或当前正在 processing 的 case 显示 spinner + `排查中`。
  - `NEED_HUMAN_CONFIRMATION` / `DONE` / `FAILED` 且本机未查看最新 `updated_at` 的 case 显示 dot + `待查看`。
  - 点击 case 后写入 localStorage，清除待查看。
- 已启动 `http://127.0.0.1:19091/web`，Browser 1600x950 验证通过。
- 已清理验证用 MySQL fixture。
- 已运行 `node -e` inline script parse、`git diff --check`、`make test`、`make secret-scan`。
- 已提交并推送：`5cdeede P-2026-042 add web case status indicators`。

下一步：

- 无；本 Program 已完成。

风险：

- 如果只用 localStorage，跨设备/跨浏览器已读不共享；本轮先按本地工作台体验处理。
