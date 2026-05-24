# HANDOFF

## 当前目标

完成真实 Qwen + health-food 本地全链路复验，提交 ai-troubleshooter 变更；health-food 外部仓库另有一处 readonly 登录态修复待提交。

## 已完成

- health-food 本地服务已启动并验证 readonly healthz/user profile/recommendation status。
- Go Gateway 已以 `CONNECTOR_MODE=http` 指向 health-food readonly endpoint，工具注册 14 个。
- Python Agent Platform 已以 `AI_MODEL_PROFILE=qwen` 启动，`/healthz` 显示 Qwen provider/model。
- API 覆盖经验命中、经验未命中、真实 Gateway 查 DB、uid 不存在、AI 配额、debug-only 本地代码检查。
- 浏览器真实打开 `http://127.0.0.1:19091/web` 并提交 `case_20260525_000035`，走 Gateway 得到 `source_date_mismatch`。
- MySQL 已反查 case/message/decision/context/tool audit。
- 已修复本 Program `ERRORS.md` 中 E2-E7；health-food E1 修复在外部仓库。
- 本轮启动的 health-food、Go Gateway、Python Agent Platform 已停止；18080/18081/19091 均无监听进程。

## 证据路径

- 索引：`programs/P-2026-040-real-qwen-health-food-full-flow/EVIDENCE.md`
- 本地原始证据：`programs/P-2026-040-real-qwen-health-food-full-flow/evidence/`，已被 `.gitignore` 忽略，不提交。

## 已运行命令

- `mvn -pl health-food-srv -am -DskipTests package`
- `curl` health-food readonly healthz/user profile/recommendation status
- `curl` Gateway `/tools` 和 `/tools/get_health_food_recommendation_status/invoke`
- `curl` Agent Platform `/healthz`、`/api/v1/chat`、`/api/v1/cases/{case_no}`
- Browser Web 提交并截图
- `PYTHONPATH=apps/agent-platform:apps/decision-engine .venv/bin/python -m unittest apps/agent-platform/tests/test_classifier.py apps/agent-platform/tests/test_service_helpers.py apps/decision-engine/tests/test_engine.py`
- `go test ./internal/gateway`
- `make test`
- `make secret-scan`
- `git diff --check`
- health-food `git diff --check`
- health-food `JAVA_HOME=$(/usr/libexec/java_home -v 23) mvn -pl health-food-srv -am -DskipTests package`
- MySQL case/message/decision/context/tool audit 查询

## 工作树/提交

- ai-troubleshooter：有代码、测试、Program、`.gitignore` 和 `HANDOFF.md` 变更，尚未提交。
- health-food：`health-food-srv/src/main/java/com/meow/pas/health/food/srv/gateway/filter/AuthInterceptorFilter.java` 有一处 readonly exclude 修复，尚未提交。

## 下一步

1. 分别提交 ai-troubleshooter 和 health-food 修复，并按仓库策略推送。

## 风险/阻塞

- 不要提交 `programs/P-2026-040-real-qwen-health-food-full-flow/evidence/` 下的原始 JSON、截图和 MySQL 输出。
