# Security Policy

ai-troubleshooter 面向业务排障场景，默认假设 Agent 不可信、Gateway 可信。任何可能扩大查询权限、泄露敏感数据、绕过审计或造成下游过载的问题都应视为安全问题。

## 支持版本

当前只维护 `main` 分支。

## 报告漏洞

请不要在公开 issue 中披露漏洞细节。优先使用 GitHub Security Advisories 私下报告；如果仓库未开启该能力，请通过维护者公开资料中的私有联系方式报告。

报告时请尽量包含：

- 受影响版本或 commit。
- 复现步骤。
- 影响范围。
- 是否涉及真实凭证、生产数据或敏感日志。
- 建议修复方向。

## 安全基线

- 生产环境必须开启 `GATEWAY_AUTH_ENABLED=true` 和 `CONTROL_API_AUTH_ENABLED=true`。
- `APP_ENV=prod` 时必须配置 Lark verification token、allowed chats、Gateway token 和控制面 token。
- Lark encrypted callback 如在飞书/Lark 后台开启，必须同步配置 `LARK_ENCRYPT_KEY`。
- 所有业务 connector 必须只读、限流、限时、脱敏、留审计。
- 不允许提交真实密钥、生产 DSN、私有证书或包含用户隐私的样例数据。
- 原图默认不持久化；如要留存图片，必须接组织内对象存储、访问控制、保留周期和数据分级流程。
