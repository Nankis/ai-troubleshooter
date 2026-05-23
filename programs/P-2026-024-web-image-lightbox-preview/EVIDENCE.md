# EVIDENCE

## 索引

| 编号 | 类型 | 说明 |
| --- | --- | --- |
| EV-T1-001 | code | `web/static/index.html` 增加 `imageLightbox`、缩略图放大按钮和 `openImagePreview/closeImagePreview`。 |
| EV-T2-001 | code | 放大层支持遮罩、关闭按钮和 Esc 关闭；图片列表变化时关闭预览并释放 object URL。 |
| EV-T3-001 | unit | `web/static/index.html` 内嵌 JS 语法解析通过；`go test ./...` 通过；`git diff --check` 通过。 |
| EV-T3-002 | browser | 本地启动 dev-server 后，独立 Chrome 粘贴图片、点击缩略图、确认大图打开；关闭按钮、Esc、遮罩关闭均通过。 |

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
  "afterPaste": {"defaultPrevented": true, "previews": 1, "fileCount": "1", "hasItems": true, "previewButton": true},
  "afterOpen": {"hidden": false, "srcIsBlob": true, "width": 722, "height": 422, "focusedClose": true},
  "afterCloseButton": true,
  "afterEscape": true,
  "afterBackdrop": true
}
```

截图：`/tmp/ai-troubleshooter-image-lightbox-open.png`
