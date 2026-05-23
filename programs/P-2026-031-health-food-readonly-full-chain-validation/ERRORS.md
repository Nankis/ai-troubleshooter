mistake_count: 3

incidents:
  - time: "2026-05-24"
    type: code
    summary: "health-food logs SQL 使用 Java text block formatted 时没有转义 DATE_FORMAT 的 `%Y/%m/%d`。"
    impact: "直连日志接口返回 BAD_REQUEST `Conversion = 'Y'`。"
    fix: "将 DATE_FORMAT 百分号改为 `%%Y` 等转义，再重启服务复验通过。"
    prevention: "凡在 Java `.formatted()` 字符串里写 MySQL DATE_FORMAT，必须转义 `%` 或改用字符串拼接。"
  - time: "2026-05-24"
    type: validation
    summary: "Web 验证发现用户明确写 `2026-05-23` 时，决策层仍按当天 `2026-05-24` 查询。"
    impact: "推荐不准确 case 会错查日期，可能给出错误结论。"
    fix: "规则抽取新增 `abnormal_date`；health-food 日级工具在推荐/token 等场景按显式日期的整天窗口查询。"
    prevention: "涉及日期的 Web 验收必须包含显式日期 case，并从工具摘要核对 `recommendation_date/start_time/end_time` 是否匹配用户输入。"
  - time: "2026-05-24"
    type: data-model
    summary: "Web 新建 case 时主表 `uid` 初始为 `web_user`，真实业务 uid 只存在于实体表和决策日志。"
    impact: "从 case 列表或 MySQL 主表看，容易误以为业务 uid 是反馈人 uid。"
    fix: "决策层抽取实体后，将 `uid/user_id` 回写到 `tb_troubleshoot_case.uid`，保留 reporter_user_id 表达反馈人。"
    prevention: "验证落库时必须同时检查 case 主表、实体表和消息表，避免业务 uid 与反馈人 uid 混淆。"

repeat_rules:
  - "真实接口验证失败时，必须先修复并复验，不能把失败项写成通过。"
  - "用户提供了明确日期时，显式日期优先级必须高于“今日/今天”等相对词。"
  - "case 主表 `uid` 表示业务 uid，反馈人必须使用 `reporter_user_id`。"
