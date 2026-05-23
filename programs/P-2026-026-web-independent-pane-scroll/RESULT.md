# RESULT

已完成 Web 工作台三栏独立滚动：

- `html/body/.app` 锁定为 viewport 高度并禁止外层滚动。
- 左侧 `.left-scroll` 独立滚动。
- 中间 `.conversation` 独立滚动，topbar 和 composer 保持固定。
- 右侧 `.right` 独立滚动。
- 三个滚动容器均使用 `overscroll-behavior: contain`，减少滚动穿透。

本地验证已通过：

- 内嵌 JS 语法解析。
- `go test ./...`
- `git diff --check`
- 独立 Chrome 实际滚动左/中/右，确认只有目标容器滚动，`window.scrollY` 始终为 0。

当前本地 dev-server 已在 `http://127.0.0.1:18088/web` 运行。
