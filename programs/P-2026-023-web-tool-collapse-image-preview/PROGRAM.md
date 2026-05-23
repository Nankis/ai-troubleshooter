# P-2026-023 Web Tool Collapse And Image Preview

## 背景

Web 工作台左侧 Gateway tools 已按服务分组，但服务组不能折叠；图片选择或粘贴后只显示数量，用户无法确认当前准备提交的是哪张截图。

## 目标

- 左侧 Gateway tools 按服务分组后支持折叠和展开。
- 图片选择或粘贴后在输入框内展示缩略图。
- 支持从待上传列表里移除单张图片。
- 保持原有 `/web/api/chat` multipart `images` 上传链路不变。
- 实际启动 Web 页面验证 UI 行为。

## 非目标

- 不修改 Gateway tools API。
- 不把折叠状态写入后端或浏览器持久化。
- 不改图片 OCR / Vision 后端识别链路。
