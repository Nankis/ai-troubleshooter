# P-2026-013 Local Code Intelligence

## 背景

当前 Local Code Agent 只做关键词级文件命中，适合证明“本地代码里有相关词”，但不足以回答“入口、任务、服务实现之间的调用关系”。用户要求继续加入 AST / call graph / LST 类能力。

## 目标

- 将本地代码检查从 keyword hit 升级为结构化代码智能。
- 输出符号索引、语言结构命中和调用边证据。
- 保持 debug-only、allowlist、denylist 和不返回源码片段的安全边界。
- 不引入重依赖，先实现轻量语言分析器；后续可替换为 tree-sitter / LSP。

## 非目标

- 不自动修改业务仓库。
- 不返回源码片段或敏感配置内容。
- 不把本地代码证据当作生产事实，只作为 Gateway 证据不足时的调试辅助。
