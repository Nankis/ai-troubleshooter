# ERRORS

mistake_count: 4

## Incidents

### E1：secret scan 对测试假值误报

- 现象：`--mode all` 命中已有单测里的 `test-key` / `vision_key`。
- 修复：允许测试/fixture/dummy 语义；同时把 `vision_key` 改成 `dummy-vision-key`。
- 防复发：测试假值统一使用 `dummy-*`、`test-*`、`fixture-*`。

### E2：Qwen 分类漏掉 `issue_domain`

- 现象：真实 Qwen text smoke 返回 issue type，但 `issue_domain` 为空，流程停在补充信息。
- 修复：OpenAI-compatible LLM 客户端增加规则基线 fallback；模型报错、漏关键字段或返回空实体时不阻断排障。
- 防复发：模型输出永远视为候选信号，生产关键字段必须有 deterministic fallback。

### E3：分钟级异常时间无法被 RFC3339 解析

- 现象：`2026-05-21T20:03+08:00` 导致工具查询时间窗回退到当前时间。
- 修复：规则抽取将分钟级时间标准化为 `2026-05-21T20:03:00+08:00`。
- 防复发：所有时间实体进入工具前必须标准化到秒级、带 timezone。

### E4：OCR 总结里的“延迟”抢占了 high mismatch 类型

- 现象：图片 smoke 中，Qwen-VL 的建议文本包含“延迟”，规则分类误成 `延迟`。
- 修复：K线分类规则中 `最高/最低/high/low` 优先于 `延迟`。
- 防复发：规则分类优先匹配用户明确症状，再匹配模型推测原因。

## Repeat Rules

- 任何 API key、密码、token 只进环境变量，不写文件。
- push 前必须运行 secret scan。
- 本轮不回写旧 Program。
- 模型输出不能单点决定生产排障路径，必须有规则/约束兜底。
- 工具时间窗必须来自标准化后的异常时间，不能静默回退当前时间。
