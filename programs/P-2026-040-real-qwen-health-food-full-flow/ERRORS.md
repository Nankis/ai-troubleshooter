# Errors

## E1. health-food readonly controller 被登录态拦截

- 现象：本地 health-food 已启动，但直接调用 `/food-health/v1/readonly/health-food/user/profile` 返回 `用户未登录`。
- 根因：health-food 自定义 `AuthInterceptorFilter` 没有把 `/food-health/v1/readonly/**` 加入 exclude list。`@SaIgnore` 对当前自定义拦截链不足以保证绕过登录态。
- 修复：在 health-food `AuthInterceptorFilter` 中加入 readonly 路径排除；readonly controller 自己仍校验 `troubleshooter.readonly.api-key`，不会放开匿名访问。
- 验证：重新编译并重启 health-food 后，正确 readonly token 返回 200，错误 token 返回 401。

## E2. Qwen issue_domain 输出不稳定导致 specialist 路由错误

- 现象：真实 Qwen 抽取同一类 health-food 问题时，`issue_domain` 曾返回 `health-food`，也曾返回 `token_usage`；`issue_type` 曾把稳定分类泛化成 `数据异常`、`数据准确性问题`、`bug`。原合并逻辑会直接覆盖规则识别结果，导致 HealthFood Agent 路由或平台经验召回不稳定。
- 根因：`merge_llm_result` 信任模型 domain/issue_type 原文，没有做领域别名归一化，也没有保护规则已命中的业务分类。
- 修复：新增 issue domain 归一化；模型返回未知领域时，如果规则已识别到领域，则不覆盖规则领域；规则已命中 `health_food/kline/asset` taxonomy 时，不用模型泛化 issue_type 覆盖。
- 验证：新增 `apps/agent-platform/tests/test_classifier.py`，覆盖 `health-food -> health_food`、未知 `token_usage` 不覆盖 `health_food`、泛化 `数据异常` 不覆盖 `餐食数据异常`。

## E3. 本地代码 debug 触发字段缺少规则抽取

- 现象：Local Code Agent 需要 `debug_local_code=true` 和 `gateway_evidence_status=insufficient`，但原规则分类器不抽取这些显式字段，完全依赖模型猜测。
- 根因：debug-only local code 入口缺少确定性实体抽取。
- 修复：规则抽取支持 `debug_local_code`、`gateway_evidence_status`、`tool_evidence_status`、`evidence_status`、`service_name`、`repo_hint`、`suspect_area` 的 `key=value` / `key:value` 形式。
- 验证：新增单测覆盖显式 debug 字段抽取。

## E4. 显式日期被 Qwen 抽成 `date`，没有传给 Gateway

- 现象：`2026-05-23 推荐数据不准` 的第一次真实排查调用了 health-food Gateway，但 `get_health_food_recommendation_status` 实际查了 `2026-05-25`，结论错误地说目标日期没有餐食。
- 根因：规则层只处理“今天/今日”，Qwen 抽取出的 `date=2026-05-23` 没有归一为 health-food 工具需要的 `recommendation_date`。
- 修复：规则层抽取 ISO 日期；merge 阶段把 `date/recommend_date/recommendDate/day` 归一到 `recommendation_date`。
- 验证：新增日期归一单测；重跑 case 后 Gateway 入参包含 `recommendation_date=2026-05-23`，health-food 返回 `job_status=source_date_mismatch`。

## E5. Gateway 摘要泄漏精确 token 余额

- 现象：`available_tokens` 数据字段已脱敏，但 `get_health_food_ai_quota` 的 `summary` 仍拼接了精确 token 余额。
- 根因：Gateway handler 手写摘要时直接使用 `result.AvailableTokens`。
- 修复：Gateway 摘要改为 `available_tokens=<redacted>`；Python `_mask` 增强 `tokens=` 文本脱敏，并对 context ledger summary 也做 mask。
- 验证：新增 `TestHealthFoodAIQuotaSummaryDoesNotExposeTokenBalance` 和 Python `_mask` 单测；重跑配额 case 后 tool evidence 和 MySQL ledger 均显示 `<redacted>`。

## E6. Web 端“查真实数据”被经验库短路

- 现象：浏览器提交“Web端验收：请查真实数据”时，命中高置信平台经验后直接回答，没有走 Gateway。
- 根因：`_requires_realtime` 只识别今日、现在、不准等词，没有把显式日期和“查真实数据/网关/数据库”作为实时查证信号。
- 修复：显式日期、查真实数据、实际数据、查 DB、Gateway/网关等文字均触发 realtime gate，高置信经验只作为候选，不再短路。
- 验证：新增 `_requires_realtime` 单测；浏览器重跑 Web case 后调用 6 个 Gateway 工具，并得到 `source_date_mismatch`。

## E7. 本地代码辅助回复没有给出可操作定位

- 现象：Local Code Agent 已命中 8 个文件、符号和调用边，但最终回复只说命中数量，业务方看不到优先文件和行号。
- 根因：`local_code_inspection` 分支复用了通用 `_need_human` 文案，丢掉了 report evidence 的可读摘要。
- 修复：新增 local-code 专用回复，列出 top 文件、行号、命中词和分析模式，仍不返回源码片段。
- 验证：重跑 local code case，回复包含 `FoodServiceImpl.java`、`MealDataFingerprintUtil.java` 等相对路径和行号。
