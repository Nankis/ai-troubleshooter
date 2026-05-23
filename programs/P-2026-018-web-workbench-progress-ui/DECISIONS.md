# DECISIONS

## D1. 保持纯 HTML/CSS/JS

当前项目强调开箱即用和私有化部署，Web 工作台继续使用 Go embed 静态页面，不引入 npm 构建链路。

## D2. 异步通过 `async=1` 增量启用

原同步 `POST /web/api/chat` 保持兼容，UI 使用 `async=1` 立即拿到 case，然后轮询状态。

## D3. 进度来自 AI decision logs

右侧进度由 `classify_issue`、`extract_entities`、`required_fields_check` 等真实决策日志映射，避免前端假进度。

## D4. 删除经验使用软删除

Web 删除知识时将 `knowledge_status` 置为 `deleted`，默认列表隐藏，避免误删平台沉淀。
