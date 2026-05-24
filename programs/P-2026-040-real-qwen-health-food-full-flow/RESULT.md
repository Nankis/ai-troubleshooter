# RESULT

## 验收结论

本轮完成 L3 本地真实依赖验收：真实本地 health-food、真实 Go Gateway、真实 Python Agent Platform、真实 Qwen 文本模型、真实 MySQL 全部跑通。

关键验收 case：

- `case_20260525_000026`：平台经验命中，未调用 Gateway。
- `case_20260525_000028`：API 链路查真实 health-food DB，定位 `source_date_mismatch`。
- `case_20260525_000030`：AI 配额查询，判断“接近上限但还能继续使用”为正常，token 余额脱敏。
- `case_20260525_000031`：uid 不存在，要求反馈者确认正确 uid。
- `case_20260525_000033`：Gateway 证据不足时，本地代码辅助定位到 health-food 相对文件和行号。
- `case_20260525_000035`：浏览器 Web 端真实提交，最终走 Gateway 得到 `source_date_mismatch`。

## 已修复问题

- health-food readonly endpoint 被用户登录态拦截。
- Qwen taxonomy 输出不稳定导致路由/经验召回不稳定。
- 显式日期没有传给 Gateway 的 `recommendation_date`。
- Gateway 配额摘要泄漏 token 余额。
- “查真实数据”被平台经验库短路。
- 本地代码检查回复缺少可操作文件和行号。

## 证据

详见 `EVIDENCE.md`。本地原始 JSON、MySQL 查询结果和截图在 `programs/P-2026-040-real-qwen-health-food-full-flow/evidence/`，该目录不提交到 Git。

## 提交

- ai-troubleshooter：实现提交和收尾状态提交均已推送 `origin/main`。
- health-food：`31657fe`，已推送 `origin/feature/P-2026-009-health-food-readonly`。

## 残留风险

- 本轮未接真实 Lark/飞书。
- 本轮不连接生产环境，不代表 L4 生产只读验收。
- health-food 外部仓库有一处 readonly 登录态排除修复，需要在 health-food 分支单独提交。
- 本地代码辅助仍是 debug-only 线索，不能替代 Gateway/DB/日志证据。
