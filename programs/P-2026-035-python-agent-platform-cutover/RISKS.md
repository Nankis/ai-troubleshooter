# Risks

- Python Agent Platform 初版需要复用现有 Web 静态页面，其 UI 交互协议必须兼容 `/web/api/*`。
- 当前 Python MySQL 访问依赖 PyMySQL；本地环境若未安装，需要按文档安装。
- Go legacy 代码仍在仓库中，后续需要单独 Program 做删除或冻结策略。
- 真实 Lark/飞书外部送达和真实外部 LLM 凭据不在本轮默认验证范围内，不能声明 L4；本轮只声明本地 HTTP encrypted callback 和 local_rules smoke。
