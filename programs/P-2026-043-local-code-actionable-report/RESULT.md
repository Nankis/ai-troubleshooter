# RESULT

## 结果摘要

- 已把本地代码辅助排查从“路径命中列表”升级为开发者可行动报告。
- Web/API 回复现在包含相对文件、具体方法/符号、行范围、命中行、可疑点、下一步核对建议和有界脱敏代码摘录。
- 决策日志只记录压缩 evidence，不把代码摘录原文写入 `output_snapshot_json`。

## 变更范围

- `apps/decision-engine/decision_engine/local_code.py`：增加 query 降噪、CamelCase 拆词、生产路径排序、方法范围识别、有界代码摘录、疑点和 follow-up 生成。
- `apps/agent-platform/agent_platform/service.py`：本地代码回复改为多行定位报告；决策日志增加 local_code evidence 压缩。
- 单测覆盖 LocalCode evidence、Verifier 检查、Web 回复 helper 和日志压缩。
- README / architecture / runbook / lessons 同步说明本地代码 debug-only 边界。

## 验证摘要

- 单测：`apps/decision-engine/tests/test_engine.py` 18 tests OK；`apps/agent-platform/tests/test_service_helpers.py` 3 tests OK。
- 回归：`make test`、`make secret-scan`、`git diff --check` 均通过。
- L3 API：Agent Platform 用 MySQL + 真实本地 health-food 源码映射启动，`case_20260525_000049` 返回可操作代码报告。
- L3 DB：MySQL 决策日志存在压缩 local_code evidence，`code_excerpt` 原文不进入 `orchestrator_plan`。
- Web：in-app browser 打开并点击 `case_20260525_000049`，页面包含 `FoodServiceImpl.generateDailyFoodRecommendWithFingerprint`、`todayRecommend.getMealDataFingerprint`、`RecommendFoodJob.refreshDailyRecommend` 和 `FoodServiceImpl.java:486-499`。

## Commit

- `P-2026-043 make local code reports actionable`

## 残留风险

- 本轮不是完整 LSP/LSIF；跨模块调用解析仍是 lightweight，复杂泛型/运行时注入关系可能需要后续接 LSP/LSIF 提升精度。
- 本地代码线索仍不是生产证据，最终判断必须结合 Gateway/DB/日志。
