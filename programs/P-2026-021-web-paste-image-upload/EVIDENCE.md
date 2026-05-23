# EVIDENCE

## 索引

| 编号 | 类型 | 说明 |
| --- | --- | --- |
| EV-T1-001 | code | `web/static/index.html` 增加 `paste` 事件、clipboard image 提取、`DataTransfer` 合并和计数刷新。 |
| EV-T2-001 | code | 提交逻辑未新增字段，仍使用 `for (const file of el.images.files) data.append("images", file)`。 |
| EV-T3-001 | runtime | 本地启动 dev-server 后，把 PNG 放入 macOS 剪贴板，在 Chrome 打开 `http://127.0.0.1:18088/web`，输入区 Cmd+V 后文件计数从 0 变为 1，toast 显示 `已粘贴 1 张截图`。 |
| EV-T3-002 | runtime | 点击发送后，后端 case 消息包含 `图片识别：image_key=pasted-20260523133616-1.png 已下载，media_type=image/png`，说明粘贴图片进入 multipart 链路。 |
| EV-T3-003 | unit | `go test ./internal/llm ./internal/decisionbaseline ./internal/webchat` 通过。 |

## 启动命令

```text
APP_ENV=dev HTTP_PORT=18088 DB_DRIVER=memory CONNECTOR_MODE=mock GATEWAY_AUTH_ENABLED=false CONTROL_API_AUTH_ENABLED=false LLM_PROVIDER=local_rules VISION_PROVIDER=local_rules go run ./cmd/dev-server
```

## 说明

Codex in-app Browser 的虚拟剪贴板不能发送图片粘贴事件。本轮使用系统级 macOS 剪贴板 + Chrome 进行真实 Cmd+V 验证。
