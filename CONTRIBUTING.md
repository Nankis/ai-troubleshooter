# Contributing

感谢你愿意改进 ai-troubleshooter。这个项目的核心目标是把业务工单排障做成可审计、可控制、可沉淀的开箱即用框架。

## 开发环境

- Go 1.24 或更高版本。
- Python 3.13，用于后续 Python Decision Engine 开发。
- Docker 可选，用于本地镜像验证。
- MySQL 是完整验收的必备依赖；默认 `DB_DRIVER=mysql` 且必须配置 `DB_DSN`。只有显式 `DB_DRIVER=memory` 才允许做一次性本地 smoke。

```bash
go version
make test
```

## 开发流程

1. 先阅读 `README.md`、`docs/deployment-checklist.md` 和相关设计文档。
2. 较大改动建议在 `programs/P-*` 下记录 scope、任务、风险、证据和结果。
3. 小步提交，每一步都保持可运行。
4. 改 DDL 时必须同时更新 migration、Go model、store、API、文档和测试。
5. 提交前运行：

```bash
git status --short
git diff --check
make test
```

## 接入业务接口

业务方接入时优先实现 `docs/ai-connector-integration.md` 中定义的只读 adapter。不要让 Agent 直接拿生产 DB、Redis、日志平台或写权限。确实需要直查 DB 时，必须使用预注册 SQL 模板、参数化查询、read replica、强制 timeout 和 limit。

## 安全要求

- 不提交真实 token、密钥、账号密码、cookie、私有证书或生产 DSN。
- 不新增写生产数据的工具。
- 不绕过 Gateway 的鉴权、scope、限流、审计和脱敏。
- 新增工具必须默认只读，并补充参数边界、审计字段和测试。
- 安全漏洞请按 `SECURITY.md` 私下报告，不要直接发公开 issue。

## PR 期望

PR 描述请包含：

- 改动目的。
- 主要实现点。
- 影响范围和风险。
- 验证命令和结果。
- 如涉及接口或 DDL，附上兼容性说明。
