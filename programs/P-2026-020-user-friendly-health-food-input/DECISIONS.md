# DECISIONS

## D1: health-food 不强制用户补充异常时间

health-food 的用户资料、AI 配额、推荐状态等工具可以基于 uid 和默认时间窗口先查证据。只有没有 uid 或无法判断异常现象时才追问。

## D2: 默认按北京时间处理自然语言日期

用户说“今天/今日”时，系统按北京时间 UTC+8 解释。追问时只说明“默认按北京时间（UTC+8）理解”，不出现 `timezone` 或 `Asia/Shanghai`。

## D3: 点查工具不接收日窗口

`get_health_food_user_profile`、`get_health_food_ai_quota`、`get_similar_cases` 不需要 `start_time/end_time`。日窗口只传给支持 24h 范围的 meal/recommendation 工具；日志查询保持 30 分钟受控窗口。
