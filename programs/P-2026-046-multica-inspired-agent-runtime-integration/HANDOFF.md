# HANDOFF

当前目标：借鉴 Multica 的 Runtime / Task lifecycle，为 ai-troubleshooter 增加 Agent Run 生命周期和 Local Runtime 抽象。

已完成：

- 建立 Program。
- 决策：不引入 Multica 依赖，不改 Go Gateway 边界；Agent Run 属于平台数据。
- 新增 migration `008_agent_runtime_runs.sql`，包含 `tb_troubleshoot_agent_runtime`、`tb_troubleshoot_agent_run`、`tb_troubleshoot_agent_run_event`。
- Python Agent Platform repository/service/server 已支持 runtime 注册、心跳、列表，以及 case agent_runs 查询。
- `process_case()` 主路径已写入 supervisor run events，并把 Knowledge/specialist/Verifier report 转为子 run。
- Web 右侧进度增加 Agent Runs 展示。
- README、架构 ADR、本地运行手册、业务接入手册已更新 Multica 借鉴边界和 API。
- 已做过 FastAPI targeted unit、migration 语法、本地 MySQL 008 应用、临时 Agent Platform chat API 落库验证。
- 最终 `make test`、`make secret-scan`、`git diff --check` 通过。
- 当前工作树启动 Agent Platform 后，`/api/v1/chat` 返回 4 个 Agent Run；MySQL 确认 `case_20260525_000051` 有 4 个 run、12 个 event。

下一步：

- 如需继续演进，下一步另起 Program 接入真实 local runtime daemon 或多 agent 并行调度。

风险：

- 本轮不实现完整 daemon，也不执行代码修改；只做可观测生命周期和后续 runtime 接入契约。
- 临时 chat API 验证使用 `local_rules`，只证明生命周期落库，不代表真实大模型排障验收。
