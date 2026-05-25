# ERRORS

## E1. Codex CLI schema 参数不兼容

- 现象：真实 `codex exec` 探针最初使用通用 `--output-schema`，Codex CLI 返回 `invalid_json_schema`，要求 object schema 的 `additionalProperties=false`。
- 修复：移除通用 schema 参数，保留 `--output-last-message`，由平台解析最后一条 JSON object；`--ask-for-approval never` 只在当前 CLI help 支持时追加。
- 防复发：本地 agent provider 不能只靠 fake 命令验收；每次改 CLI 参数都必须跑真实 provider 探针。

## E2. Agent Run 模型来源字段不清晰

- 现象：`llm_decision_agent` payload 已记录 `provider=local_agent`、`local_provider=codex`，但 `tb_troubleshoot_agent_run.model_provider/model_name` 仍显示平台主模型 `local_rules/rules-v1`。
- 修复：根据 agent report observations 覆盖 `llm_decision_agent` run 的模型字段，真实记录为 `local_agent/codex`。
- 防复发：只要输出给用户或落库用于观测，就不能只把真实来源藏在 payload 深层字段。

本 Program 暂无新增错误复盘。
