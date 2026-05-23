# EVIDENCE

## 索引

| 编号 | 类型 | 说明 |
| --- | --- | --- |
| EV-T1-001 | code | `internal/webchat/handler.go` 支持 `GET/PUT /web/api/knowledge/{id}`，编辑时合并已有字段。 |
| EV-T2-001 | code | `web/static/index.html` 增加经验详情弹层，展示典型现象、步骤、常见原因、工具和统计信息。 |
| EV-T3-001 | code | `web/static/index.html` 增加编辑按钮，复用录入表单进入编辑模式并调用 PUT 保存。 |
| EV-T4-001 | unit | `web/static/index.html` 内嵌 JS 语法解析通过；`go test ./...` 通过；`git diff --check` 通过。 |
| EV-T4-002 | browser | 本地启动 dev-server 后，独立 Chrome 实际新增、预览、编辑和删除经验均通过。 |

## 验证摘要

命令：

```text
node -e "...new Function(extracted web script)..."
go test ./...
git diff --check
APP_ENV=dev HTTP_PORT=18088 DB_DRIVER=memory CONNECTOR_MODE=mock GATEWAY_AUTH_ENABLED=false CONTROL_API_AUTH_ENABLED=false LLM_PROVIDER=local_rules VISION_PROVIDER=local_rules /tmp/ai-troubleshooter-dev-server
open -na "Google Chrome" --args --user-data-dir=/tmp/ai-troubleshooter-chrome-profile --remote-debugging-port=19333 --window-size=1400,900 http://127.0.0.1:18088/web
```

浏览器实际结果：

```json
{
  "afterCreate": {"rows": 1, "title": "测试 行情 · k线数据不准", "formOpen": false},
  "previewBeforeEdit": {"hidden": false, "stats": "命中 1 次 · 置信度 70%"},
  "editLoaded": {"id": "1", "submit": "保存修改"},
  "afterEdit": {"rows": 1, "title": "测试 行情 · k线数据不准 已编辑", "formOpen": false},
  "previewAfterEdit": {"title": "测试 行情 · k线数据不准 已编辑"},
  "afterDelete": {"rows": 0, "emptyText": "暂无沉淀经验", "modalHidden": true}
}
```

截图：`/tmp/ai-troubleshooter-knowledge-preview-edit.png`
