# DECISIONS

## D1：优先实现可运行的跨模块语义解析

先在现有 Local Code Agent 内完成 receiver type、接口实现关系和 resolved call edge，保证今天能在 health-food 仓库上跑出有用证据。

## D2：tree-sitter / LSP / LSIF 作为后端，不改变证据契约

无论底层后端来自轻量扫描、tree-sitter、LSP 还是 LSIF，Local Code Agent 对外仍只返回相对路径、符号、调用边、解析结果和行号。
