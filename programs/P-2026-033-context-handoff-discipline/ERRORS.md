mistake_count: 1

incidents:
  - time: "2026-05-24"
    type: workflow
    summary: "P-2026-032 完成时没有同步创建或更新 HANDOFF.md。"
    impact: "上下文压缩后接手者需要从聊天记忆恢复，风险高。"
    fix: "补 P-2026-032 HANDOFF，并把压缩前/恢复后 HANDOFF 规则写入 AGENTS.md 和 programs/README.md。"
    prevention: "Program 没有最新 HANDOFF.md 时，不允许宣称完成、暂停或切换方向。"

repeat_rules:
  - "看到自动压缩提示时，先写 HANDOFF，再继续压缩后的工作。"
  - "压缩后恢复时，先读 HANDOFF，再执行任何代码或文档改动。"
