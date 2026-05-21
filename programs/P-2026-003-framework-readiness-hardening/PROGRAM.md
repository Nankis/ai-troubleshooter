# P-2026-003 Framework Readiness Hardening

## 背景

用户要求再次检查一期排障 Agent 框架是否可以作为业务方开箱使用。上一轮已补齐 Gateway HTTP 鉴权、身份绑定和限流，本轮继续按准交付审计检查控制面接口、生产配置 fail-closed、policy 边界和验证流程。

## 目标

- 补齐控制面 API 的内部 Bearer 鉴权。
- 生产环境配置缺失时 fail-closed。
- 修复 policy 中 Lark 群权限缺少 `chat_id` 时的放行边界。
- 更新部署文档和测试。

## 非目标

- 不引入真实 Redis Stream。
- 不实现真实 Lark 签名解密和消息发送。
- 不替代公司 API Gateway / service mesh 的 mTLS 和分布式限流。

## 验收标准

- `APP_ENV=prod` 下 Gateway 未启用鉴权或无 token 时启动失败。
- `APP_ENV=prod` 下控制面 API 未配置 token 时启动失败。
- 控制面 API 未带 token 返回 401。
- agent 配置了允许群时，请求缺少 `chat_id` 必须 deny。
- `git diff --check`、`go vet ./...`、`make test` 通过。
