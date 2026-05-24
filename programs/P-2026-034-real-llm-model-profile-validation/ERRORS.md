mistake_count: 1

incidents:
  - time: "2026-05-24"
    type: validation
    summary: "此前把真实数据链路验证说成接近真实 Agent 验证，但决策层仍是 local_rules。"
    impact: "用户以为已经接入真实大模型，实际只是规则模板生成结论。"
    fix: "新增真实 LLM profile、严格模式和 Web 真实 LLM 验证。"
    prevention: "每次验收必须同时声明 evidence source 和 decision model；local_rules 不得写成真实 LLM Agent。"

repeat_rules:
  - "真实数据链路和真实 LLM 决策必须分开记录。"
  - "非 local_rules 的 LLM provider 默认禁止静默 fallback。"

runtime_findings:
  - time: "2026-05-24"
    type: real_llm_schema
    summary: "真实 Qwen 首次 Web 验证返回字段别名，严格模式失败。"
    fix: "OpenAI-compatible 客户端兼容中英文字段别名，并在错误中保留脱敏 raw 摘要便于定位。"
  - time: "2026-05-24"
    type: tool_selection_quality
    summary: "真实 Qwen 第二次只选择 user_profile，无法定位推荐日期错配。"
    fix: "health-food 推荐/token 场景增加最低证据工具 guardrail，并在 decision log 标记 augmented=true。"
