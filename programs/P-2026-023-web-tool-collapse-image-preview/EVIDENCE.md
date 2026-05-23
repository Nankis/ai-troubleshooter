# EVIDENCE

## 索引

| 编号 | 类型 | 说明 |
| --- | --- | --- |
| EV-T1-001 | code | `web/static/index.html` 增加 `collapsedToolServices`、折叠按钮、`aria-expanded` 和折叠样式。 |
| EV-T2-001 | code | `web/static/index.html` 增加 `imagePreviewList`、缩略图渲染、单张移除和 object URL 释放。 |
| EV-T3-001 | unit | `go test ./...` 通过；`web/static/index.html` 内嵌 JS 语法解析通过。 |
| EV-T3-002 | browser | 本地启动 dev-server 后用独立 Chrome 宽屏打开 `/web`，点击 `health-food` 服务组后 `aria-expanded=false` 且 body `display=none`；再次点击恢复 `aria-expanded=true` 且 body `display=grid`。 |
| EV-T3-003 | browser | 本地启动 dev-server 后通过 CDP 给 `#images` 设置文件，输入框内出现 1 个缩略图；点击移除后缩略图和计数归 0；模拟粘贴图片后缩略图和计数变为 1。 |

## 验证摘要

命令：

```text
node -e "...new Function(extracted web script)..."
go test ./...
APP_ENV=dev HTTP_PORT=18088 DB_DRIVER=memory CONNECTOR_MODE=mock GATEWAY_AUTH_ENABLED=false CONTROL_API_AUTH_ENABLED=false LLM_PROVIDER=local_rules VISION_PROVIDER=local_rules /tmp/ai-troubleshooter-dev-server
open -na "Google Chrome" --args --user-data-dir=/tmp/ai-troubleshooter-chrome-profile --remote-debugging-port=19333 --window-size=1400,900 http://127.0.0.1:18088/web
```

浏览器实际结果：

```json
{
  "groups": ["health-food:4", "asset-service:2", "market-service:4", "ops/logs:3", "platform:1"],
  "after_collapse": {"collapsed": true, "expanded": "false", "bodyDisplay": "none"},
  "after_expand": {"collapsed": false, "expanded": "true", "bodyDisplay": "grid"},
  "after_file_select": {"previews": 1, "fileCount": "1", "hasItems": true},
  "after_remove": {"previews": 0, "fileCount": "0", "hasItems": false},
  "after_paste": {"defaultPrevented": true, "previews": 1, "fileCount": "1", "hasItems": true}
}
```

截图：`/tmp/ai-troubleshooter-tool-collapse-image-preview.png`
