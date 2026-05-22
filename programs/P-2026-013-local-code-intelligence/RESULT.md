# RESULT

## 结论

Local Code Agent 已从关键词级检索升级为轻量代码智能检查。它仍然是 debug-only，并且只返回相对路径、命中词、符号、调用边和行号，不返回源码片段。

## 已完成

- `LocalCodeHit` 新增 `symbols`、`call_edges`、`analysis_modes`。
- Python 使用 stdlib AST 提取 class / function / call。
- Java / Go / TypeScript / JavaScript 使用轻量语言结构扫描提取 class / method / function / call。
- Agent observations 增加 `symbol_count`、`call_edge_count`、`analysis_modes`。
- health-food 真实仓库试跑能看到 `RecommendFoodJob.refreshDailyRecommend -> generateDailyFoodRecommendWithFingerprint`。
- 补充 Java 调用边和 Python AST 调用边单测。

## 后续

- 当前 analyzer 是轻量实现，不等价于完整编译器。
- 如果要跨模块精准调用链、类型解析、接口实现关系，下一步接 tree-sitter / LSP / LSIF。
