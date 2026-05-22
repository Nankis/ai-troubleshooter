# DECISIONS

## D1：先做轻量内置 analyzer

一期不引入 tree-sitter / LSP server 依赖，避免部署复杂度上升。先在 Python Decision Engine 中实现轻量符号索引和调用边，接口上保留 `analysis_modes`，后续可替换为更强 analyzer。

## D2：证据仍然不能包含源码片段

本地代码检查用于定位，不用于泄露代码内容。证据只返回相对路径、符号名、符号类型、行号和调用关系。
