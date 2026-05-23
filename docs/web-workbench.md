# Web 排障工作台

内置 Web 工作台用于不接 Lark/飞书时直接排查问题。页面入口：

```text
GET /web
GET /
```

## 布局

- 左侧：问题会话、按服务可折叠分组的已注册 Gateway tools、平台经验沉淀。
- 中间：当前问题对话和 Agent 输出。
- 右侧：决策层进度、当前状态、工具数量、知识数量和证据来源。

## 输入

- 文字输入：描述生产问题。
- 图片输入：点击 `截图` 选择本地图片，或在输入框聚焦时直接粘贴剪贴板图片。选择或粘贴后会在输入框内预览缩略图，并进入同一个 `images` multipart 字段，和手动选择图片共用后端链路。

## API

| API | 说明 |
| --- | --- |
| `GET /web/api/overview` | 返回 recent cases、已注册 tools、知识库条目。 |
| `POST /web/api/chat` | 创建或继续 case，支持 `async=1` 异步排查。 |
| `GET /web/api/cases/{case_no}` | 查询 case、消息、实体、AI decision logs、progress steps。 |
| `GET /web/api/knowledge` | 查询平台知识。 |
| `POST /web/api/knowledge` | 手动录入知识。 |
| `DELETE /web/api/knowledge/{id}` | 软删除指定知识，默认列表不再展示。 |

## 进度来源

右侧进度不是前端假状态，而是从 `tb_troubleshoot_ai_decision_log` / in-memory store 中读取：

- `classify_issue`
- `extract_entities`
- `required_fields_check`
- `knowledge_retrieval`
- `decide_next_action`
- `tool_invocation`
- `summarize_findings`

当 case 处于 `READY_TO_INVESTIGATE`、`INVESTIGATING`、`WAITING_TOOL_RESULT` 时，页面显示为排查中，并持续轮询 `/web/api/cases/{case_no}`。
