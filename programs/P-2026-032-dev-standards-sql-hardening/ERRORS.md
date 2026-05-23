mistake_count: 1

incidents:
  - time: "2026-05-24"
    type: security
    summary: "真实 health-food Python adapter 使用 f-string 拼接 MySQL SQL。"
    impact: "虽然 uid/date 有局部校验，但会形成错误示范，并在后续扩展参数时放大 SQL 注入风险。"
    fix: "改为 PyMySQL DB-API 参数绑定；新增测试防止 `mysql_query(f\"...\")` 回归。"
    prevention: "任何 DB 输入必须通过参数绑定；动态 SQL 只能拼接代码内白名单片段，并用测试证明外部输入不进入 query 文本。"

repeat_rules:
  - "不要把参数校验当作 SQL 拼接的安全理由；校验和参数绑定必须同时存在。"
  - "发现安全规范缺口时，必须同时更新 AGENTS.md、代码和测试。"
