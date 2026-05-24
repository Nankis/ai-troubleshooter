# Handoff

## 当前目标

把排障平台从 `local_rules` 验证升级为真实大模型验证，并提供 `AI_MODEL_PROFILE` / `AI_MODEL_CONFIG_FILE` 这类统一切换入口。

## 已完成

- 确认默认 `LLM_PROVIDER=local_rules`，此前 Web 结论不是大模型推理。
- 确认 `worker` 直接使用 `llm.NewRuleBasedClient()`，需要修复。
- 安全读取 health-food 本地 `application-local.yml`，实际调用 Qwen OpenAI-compatible 成功；密钥未输出。
- 实现 `AI_MODEL_PROFILE` / `AI_MODEL_CONFIG_FILE`，支持 qwen/dashscope/deepseek/moonshot/openai/local_rules。
- `cmd/dev-server` 和 `cmd/worker` 已改为 `llm.NewFromConfig(cfg.LLM)`。
- OpenAI-compatible 客户端默认严格模式，`LLM_ALLOW_RULE_FALLBACK=false` 时禁止真实 LLM 失败后静默规则兜底。
- 修复 health-food prompt/schema，兼容真实模型常见字段别名。
- 增加 health-food 推荐/token 场景最低证据工具 guardrail。
- 本地启动 health-food readonly 服务、ai-troubleshooter Web、真实 Qwen，Web UI 跑通 `case_20260524_000013`，定位 `source_date_mismatch`。

## 证据路径

- `EVIDENCE.md`
- `RESULT.md`
- `evidence/web-real-qwen-health-food-result.png`

## 已运行命令

- `MYSQL_HOST=127.0.0.1 MYSQL_PORT=3306 MYSQL_USER=root MYSQL_PASSWORD=*** MYSQL_DATABASE=ai_troubleshooter make migrate-mysql`
- `go test ./internal/config ./internal/llm ./internal/vision ./cmd/worker`
- `go test ./internal/decisionbaseline ./internal/llm ./internal/config ./cmd/dev-server`
- `make test`
- `make secret-scan`
- `git diff --check`
- health-food 本地启动：`mvn -pl health-food-srv spring-boot:run ... --server.port=18080 ...`
- ai-troubleshooter 本地启动：`AI_MODEL_PROFILE=qwen ... LLM_ALLOW_RULE_FALLBACK=false ... go run ./cmd/dev-server`

## 下一步

- 提交并推送 main。
- 结束前停止本轮启动的本地服务，除非用户要求保留。

## 工作树

- 当前有本 Program 代码、文档和截图改动未提交。
- 不要提交任何本地密钥或 health-food 配置文件。
