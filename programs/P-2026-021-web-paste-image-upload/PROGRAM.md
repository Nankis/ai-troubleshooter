# P-2026-021 Web Paste Image Upload

## 背景

Web 排障入口已支持点击选择图片，但用户更常见的操作是截图后直接复制，再在输入框粘贴。原页面没有处理 clipboard image，粘贴后不会进入 `images` multipart 上传链路。

## 目标

- 输入区支持粘贴剪贴板图片。
- 粘贴图片与手动选择图片合并到同一个 file input。
- 提交后仍走 `/web/api/chat` 的 `images` multipart 字段。
- 提交或新建会话后清空图片计数。
- 实际启动本地 Web 服务验证粘贴和提交。

## 非目标

- 不修改后端图片识别接口。
- 不引入前端框架或新的构建系统。
