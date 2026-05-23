# RESULT

## 结论

Web 排障入口已支持复制图片后直接粘贴上传。粘贴图片会并入原 `images` file input，发送后进入现有图片识别和 case 排查链路。

## 验证

- 真实 Cmd+V 后图片计数变为 1。
- 提交后 case 消息出现 `image_key=pasted-...png`。
- 相关 Go 测试通过。
