# RISKS

- 任意 MCP command 如果直接执行，会变成远程命令执行入口；本轮禁止。
- MCP tool 描述可能伪装成 readonly；本轮只把描述作为参考，最终以 allowlist、安全规则和人工发布为准。
- 没有真实 secret manager，`secret_ref` 只能引用环境变量或公司密钥系统名称。
- 单进程 dev-server reload 已覆盖本地体验，多实例生产仍需要配置发布事件或集中 registry reload。
