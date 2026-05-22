# P-2026-012 Health Food Real Adapter Verification

## 背景

用户指出上一轮 health-food 验证只使用 mock adapter 包装本地服务探活和可控故障数据，不能证明业务真实信息可靠。真实验收标准应该是服务都运行起来，并通过注册接口查到可靠证据，例如真实 DB 数据、日志或代码定位。

## 目标

- 启动本地 `health-food` 服务和排障平台依赖。
- 尽可能通过 health-food 真实注册/登录流程创建或使用测试账号。
- 建立真实 readonly adapter：不返回 mock 故障数据，证据来自 health-food 本地 DB、服务探活、日志文件或本地代码只读定位。
- 通过排障平台 Web/API 完整跑 health-food case，要求 Agent 查到真实证据。
- 记录所有无法完成的真实阻塞，不能用 mock 结果冒充真实验收。

## 非目标

- 不提交本地账号密码、token、API key 或真实业务隐私数据。
- 不改 health-food 仓库代码，除非用户明确要求。
- 不把 Agent 直接接生产 DB；本轮只使用本地测试 DB。
- 不用 mock 故障数据冒充真实业务证据。
