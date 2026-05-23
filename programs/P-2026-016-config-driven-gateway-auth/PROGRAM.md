# P-2026-016 Config Driven Gateway Auth

## 背景

用户指出当前 Gateway agent、scope、tool 授权关系主要写在代码中，不够开箱即用。业务方新增 agent 或调整权限时不应该改 Go 代码。

## 目标

- 支持通过 JSON 配置文件或环境变量配置 Gateway agents、scopes、tools、chat allowlist 和 token env。
- 保留现有默认 agent，确保本地 demo 不被破坏。
- 将 worker/dev-server/baseline runner 的 agent id 从代码常量改为配置。
- 补充测试、文档和样例配置。
- 扫描其它明显影响开箱接入的硬编码点，并记录处理结果。

## 非目标

- 本轮不做完整控制台权限管理。
- 本轮不把 Tool Registry 改成完全数据库动态注册。
- 本轮不引入 YAML 解析依赖；Gateway agent 配置先用 JSON。
