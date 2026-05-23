# HANDOFF

## 当前状态

- Program 已完成，Local Code Agent 具备轻量跨模块语义索引能力。
- 已支持 Java receiver type、接口实现关系和 resolved call edge。
- tree-sitter / LSP / LSIF 已作为 backend 配置位预留，但本轮不强制引入外部语言服务器依赖。

## 下一步

- 如要继续提升精度，优先新增真实 tree-sitter Java backend，再补 LSP/LSIF 后端读取器。
- 增加真实业务仓库的语义索引评测集，覆盖 Spring 注入、泛型、重载、跨 module Maven/Gradle 结构。
