# P-2026-020 User Friendly Health Food Input

## 背景

Web 排障入口在 health-food 图片 case 里要求用户补充 `异常发生的大概时间，并带 timezone，默认 Asia/Shanghai`。这暴露了内部字段和技术词，且不符合真实用户反馈方式。

业务用户通常只会提供类似 `uid:123456 用户反馈 今日 token消耗 数量不对` 的自然语言。系统应自行按默认北京时间理解“今日/今天”，只有必要信息真的缺失时才用业务语言追问。

## 目标

- health-food 问题不再把 `abnormal_time` 作为用户必填。
- 支持 `token消耗/数量/用量` 归类为 AI 配额异常。
- 缺信息追问不出现 `timezone`、`Asia/Shanghai` 等内部说法。
- 对“今日/今天”的 health-food 工具参数使用合适窗口，并避免把日窗口传给点查工具导致 Gateway 拒绝。
- 用真实本地 Web 服务验证用户示例能直接进入排查。

## 非目标

- 不改变 Gateway 的 scope、鉴权、限流、timeout 和时间范围约束。
- 不引入新的 LLM provider 或改动 Python agent-team。
