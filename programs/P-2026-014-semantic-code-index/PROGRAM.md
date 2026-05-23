# P-2026-014 Semantic Code Index

## 背景

Local Code Agent 已有关键词、符号和有限调用边，但要更接近真实代码排查，需要跨模块调用链、类型解析和接口实现关系。用户提出下一步接 tree-sitter / LSP / LSIF。

## 目标

- 增加跨模块调用边解析。
- 增加 Java receiver type 和接口实现关系解析。
- 输出 resolved call edge，包含被解析到的符号相对路径和行号。
- 保持本地代码证据安全边界：debug-only、allowlist、无源码片段。
- 预留 tree-sitter / LSP / LSIF backend 配置入口。

## 非目标

- 本轮不强制引入外部语言服务器或重型索引依赖。
- 不自动修改业务代码。
- 不把本地代码证据当作生产事实。
