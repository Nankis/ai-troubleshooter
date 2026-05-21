# AI 决策日志与查询限制

本系统不允许 Agent 在生产里无限自主循环。Orchestrator 采用“有限工具计划”：先分类、抽取字段、检查必要字段，信息足够后只执行一轮有上限的只读工具查询，再输出需要人工确认的排查结论。

## 决策日志

每个 case 的关键 AI 决策都会写入 `ai_decision_logs`：

- `classify_issue`：为什么判断为某个业务域和问题类型。
- `extract_entities`：抽取了哪些字段。
- `required_fields_check`：为什么追问用户或为什么开始排查。
- `decide_next_action`：为什么选择这些工具。
- `tool_invocation`：每次工具调用的入参摘要、返回状态、summary、tool call id。
- `tool_query_stopped`：达到失败上限后为什么停止继续查下游。
- `summarize_findings`：如何基于有限证据生成结论。
- `process_failure`：超时或异常时的失败原因。

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

## 失败收敛

如果分类、实体抽取、工具计划、工具调用或总结阶段发生不可恢复错误，orchestrator 会：

1. 写入 `process_failure` 决策日志。
2. 如果 investigation 已创建，将 investigation 标记为 `failed`。
3. 如果 case 还没有进入终态，将 case 标记为 `FAILED`。
4. 写入 system message，说明排查停止原因。

这样 worker 不会让同一个 case 长时间停留在 `INVESTIGATING` 或 `WAITING_TOOL_RESULT`。
