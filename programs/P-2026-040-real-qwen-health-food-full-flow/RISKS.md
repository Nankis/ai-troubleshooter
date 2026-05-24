# Risks

- health-food 本地服务可能因本机配置、端口、Redis/MySQL 或 JDK 版本启动失败。
- 真实 Qwen 调用可能因 key、网络或模型返回非 JSON 失败；失败时不能静默降级到 local_rules。
- 本地代码辅助当前不等同生产证据，只能作为 debug 辅助证据。
- 本轮不能连接生产或执行写操作，所有 health-food 调用都必须是 readonly。
