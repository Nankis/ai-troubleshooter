# Result

## 当前结果

- 新增统一模型入口：`AI_MODEL_PROFILE` / `AI_MODEL_CONFIG_FILE`，支持 `qwen`、`dashscope`、`deepseek`、`moonshot`、`openai` 和 `local_rules`。
- `cmd/dev-server` 和 `cmd/worker` 都通过 `llm.NewFromConfig(cfg.LLM)` 使用配置化模型；真实 provider 默认 `LLM_ALLOW_RULE_FALLBACK=false`，失败直接暴露。
- OpenAI-compatible 客户端增强为严格 JSON schema prompt，并兼容真实模型常见字段别名。
- health-food 推荐类问题增加最低证据工具 guardrail，避免模型只查一个工具就下结论。
- Web UI 已用真实 Qwen 跑通 `case_20260524_000013`，连接真实 MySQL、真实 health-food readonly endpoint，查到 `source_date_mismatch`。

## 验收摘要

- LLM：`model_provider=qwen`、`model_name=qwen-plus` 写入 `tb_troubleshoot_investigation`。
- Evidence：截图位于 `programs/P-2026-034-real-llm-model-profile-validation/evidence/web-real-qwen-health-food-result.png`。
- 下游证据：`get_health_food_recommendation_status` 返回 `exists=true`、`job_status=source_date_mismatch`。

## 残留风险

- 本轮真实 LLM 验证使用 health-food 本地库和本地 readonly 服务，不是生产 health-food。
- 图片真实视觉模型未在本轮重新验收；本轮配置使用 `VISION_PROVIDER=local_mock` 避免把无关图片链路混入文本决策验收。
- Go baseline 仍是 phase-0；Python agents team 仍是目标决策层方向。
