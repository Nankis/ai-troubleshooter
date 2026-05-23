# EVIDENCE

## 索引

| 编号 | 类型 | 说明 |
| --- | --- | --- |
| EV-T1-001 | code | `web/static/index.html` 将 `html/body/.app` 锁定为 viewport 高度并禁止外层滚动。 |
| EV-T2-001 | code | `.left-scroll`、`.conversation`、`.right` 独立 `overflow:auto` 且 `overscroll-behavior:contain`。 |
| EV-T3-001 | unit | `web/static/index.html` 内嵌 JS 语法解析通过；`go test ./...` 通过；`git diff --check` 通过。 |
| EV-T3-002 | browser | 本地启动 dev-server 后，独立 Chrome 宽屏验证左/中/右滚动互不影响，页面整体不滚动。 |

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
  "before": {
    "windowScrollY": 0,
    "documentClientHeight": 900,
    "documentScrollHeight": 900,
    "bodyOverflow": "hidden",
    "appOverflow": "hidden",
    "leftMax": 616,
    "middleMax": 1887,
    "rightMax": 1654
  },
  "afterLeftWheel": {"windowScrollY": 0, "left": 520, "middle": 0, "right": 0},
  "afterMiddleWheel": {"windowScrollY": 0, "left": 520, "middle": 520, "right": 0},
  "afterRightWheel": {"windowScrollY": 0, "left": 520, "middle": 520, "right": 520},
  "assertions": {
    "outerLocked": true,
    "noDocumentOverflow": true,
    "leftOnlyMovedFirst": true,
    "middleOnlyMovedSecond": true,
    "rightOnlyMovedThird": true
  }
}
```

截图：`/tmp/ai-troubleshooter-independent-pane-scroll.png`
