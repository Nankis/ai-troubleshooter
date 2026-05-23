# Risks

- health-food 仓库历史配置文件含敏感内容；本轮不修改这些文件，不在 Program/提交中新增密钥。
- 本地 `meow_pas` 数据是历史测试数据，不代表生产全量行为；但本轮验收只声明本地真实 DB 链路通过。
- logs search 当前查 `tb_ai_message_log`，不是机器日志文件；用于本地排障证据足够，生产日志仍应接日志平台或 DMS/MCP。
