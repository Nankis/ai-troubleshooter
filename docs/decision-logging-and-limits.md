# AI 决策日志与查询限制

本系统不允许 Agent 在生产里无限自主循环。Decision Layer 采用“有限工具计划”：先分类、抽取字段、检查必要字段，信息足够后只执行一轮有上限的只读工具查询，再输出需要人工确认的排查结论。当前 Go `decisionbaseline` 和 Python `decision-engine` 都必须遵守同一套限制。

Python `decision-engine` 的 Agent Team 也遵守同一规则：Supervisor 负责路由，Knowledge Agent 判断历史经验是否可直接复用，Kline / Asset Agent 只生成 Gateway 只读工具计划，Local Code Agent 仅在 debug-only 且 Gateway 证据不足时读取本地 allowlist 仓库，Verifier 最后统一做预算、去重、可用工具过滤和失败收敛。

## 决策日志

每个 case 的关键 AI 决策都会写入 `tb_troubleshoot_ai_decision_log`：

- `classify_issue`：为什么判断为某个业务域和问题类型。
- `extract_entities`：抽取了哪些字段。
- `required_fields_check`：为什么追问用户或为什么开始排查。
- `decide_next_action`：为什么选择这些工具。
- `agent_team_report`：Python Agent Team 中 Supervisor / Specialist / Verifier 的中间判断和拒绝原因。
- `local_code_inspection`：debug-only 本地代码辅助排查的 repo id、相对路径、命中词、符号、调用边、行号和拒绝原因；不能写入源码片段或本地绝对路径。
- `tool_invocation`：每次工具调用的入参摘要、返回状态、summary、tool call id。
- `tool_query_stopped`：达到失败上限后为什么停止继续查下游。
- `summarize_findings`：如何基于有限证据生成结论。
- `process_failure`：超时或异常时的失败原因。
- `process_skipped`：重复事件、重复 worker 或非入口状态为什么跳过处理。
- `process_stale_timeout`：case 卡在处理中状态超过陈旧窗口后为什么失败收敛。
- `process_stale_claim_recovered`：case 卡在 `READY_TO_INVESTIGATE` 超过陈旧窗口后为什么重新认领。

`input_snapshot_json`、`output_snapshot_json` 和 `selected_tools_json` 写入前会经过 `masking.MaskValue`，手机号、邮箱、token、secret、api key、access key、address、raw payload 等敏感值不落明文。原始 case 文本如需长期留存，应由业务方根据公司数据分级要求控制保留周期或再做列级加密。

查询：

```bash
GET /cases/{case_no}/ai-decisions?limit=100
```

case 详情接口也会带上最近的决策日志：

```bash
GET /cases/{case_no}?decision_log_limit=100
```

## 查询限制

环境变量：

```bash
MAX_INVESTIGATION_SECONDS=120
MAX_TOOL_CALLS_PER_CASE=10
MAX_TOOL_FAILURES_PER_CASE=3
DEFAULT_TOOL_TIMEOUT_SECONDS=5
```

含义：

- `MAX_INVESTIGATION_SECONDS`：单个 case 从 worker 开始处理到结束的总时间上限；超时后 case 收敛为 `FAILED`。
- `MAX_TOOL_CALLS_PER_CASE`：单个 case 最多执行的工具数；LLM 返回更多工具会被截断。
- `MAX_TOOL_FAILURES_PER_CASE`：工具失败达到阈值后停止继续调用下游，进入总结阶段并明确证据不足。
- `DEFAULT_TOOL_TIMEOUT_SECONDS`：Gateway 单次工具 handler 超时。

Gateway 侧仍有独立保护：

- HTTP Bearer 鉴权。
- agent 与 `agent_id` 绑定。
- policy 默认拒绝。
- 时间范围、limit 边界。
- agent/user/tool QPS 限流。
- 下游 readonly connector timeout。

## 幂等和重复处理保护

Lark/飞书平台可能因为网络抖动或业务接口超时重复投递同一条消息。系统按 `source + message_id` 建 case 幂等键：

- Lark/飞书 handler 在创建 case 前先查询同一 `source + message_id` 是否已被接收。
- 已接收的事件返回 `202`，响应包含 `duplicate=true` 和已有 `case_no`，不再重复入队。
- MySQL 通过 `migrations/004_case_idempotency.sql` 增加唯一索引，防并发重复创建。

Worker 侧也有第二道保护：Decision runner 只允许 `NEW`、`NEED_MORE_INFO`、`WAITING_USER_REPLY` 进入处理，并会先把 case 认领到 `READY_TO_INVESTIGATE`。如果重复 worker、重复事件或终态 case 再次触发处理，系统写入 `process_skipped` 决策日志并返回当前状态，不会再次调用下游工具。

为了避免 worker 崩溃后 case 永久卡住，Decision runner 使用 `MAX_INVESTIGATION_SECONDS * 2` 作为陈旧窗口，最小 60 秒：

- `READY_TO_INVESTIGATE` 超过窗口会被重新认领并继续处理。
- `INVESTIGATING` 或 `WAITING_TOOL_RESULT` 超过窗口会写入 `process_stale_timeout`，并收敛为 `FAILED`，避免后台任务无限占用处理中状态。

## 失败收敛

如果分类、实体抽取、工具计划、工具调用或总结阶段发生不可恢复错误，Decision runner 会：

1. 写入 `process_failure` 决策日志。
2. 如果 investigation 已创建，将 investigation 标记为 `failed`。
3. 如果 case 还没有进入终态，将 case 标记为 `FAILED`。
4. 写入 system message，说明排查停止原因。

这样 worker 不会让同一个 case 长时间停留在 `INVESTIGATING` 或 `WAITING_TOOL_RESULT`。
