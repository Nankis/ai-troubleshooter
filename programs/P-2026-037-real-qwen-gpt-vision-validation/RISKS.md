# RISKS

| 风险 | 影响 | 处理 |
| --- | --- | --- |
| 本机没有 GPT key | GPT 真实验收无法完成 | 实现 GPT provider，记录真实验收阻塞，不用 Qwen 冒充 GPT。 |
| health-food config 字段变化 | 自动读取失败 | 支持环境变量覆盖；单测覆盖当前 Spring AI YAML 结构。 |
| Vision 费用和图片 token 消耗 | 测试成本增加 | 使用小尺寸测试图，限制图片数量和大小。 |
| 模型返回非 JSON | 分类/总结失败 | prompt 约束只输出 JSON，解析器会抽取首个 JSON object；真实验收开启 fail-fast 时记录错误。 |
| Chrome extension 不允许直接 setFiles | 自动化文件上传受阻 | 用真实 Chrome 系统文件选择器手动选择同一测试图片，并把页面截图和 MySQL 证据写入 Program。 |
