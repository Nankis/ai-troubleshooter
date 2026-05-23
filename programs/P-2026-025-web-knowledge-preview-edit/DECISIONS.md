# DECISIONS

## D1: 预览用弹层，编辑复用左侧表单

经验预览需要完整内容空间，因此用 modal 展示详情；编辑复用已有录入表单，进入“保存修改”模式，减少重复表单和字段分叉。

## D2: 后端显式支持 GET/PUT

虽然底层 `UpsertKnowledgeItem` 已支持 ID 更新，但 Web API 原来只有 POST/DELETE。本轮增加 `GET /web/api/knowledge/{id}` 和 `PUT /web/api/knowledge/{id}`，让前端行为和 HTTP 语义更清晰。

## D3: 编辑时保留未暴露字段

Web 表单只编辑标题、领域、类型、典型现象、步骤、常见原因、相关工具。后端更新时先读取已有知识，再覆盖表单字段，保留置信度、历史命中次数、最后根因等未暴露字段。
