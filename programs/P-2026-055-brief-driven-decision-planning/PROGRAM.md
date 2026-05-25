# P-2026-055 Brief Driven Decision Planning

## Objective

让 Python Supervisor 的工具选择优先围绕 Brief 的目标、假设和成功标准展开，减少上下文膨胀和无关查询。

## Scope

- Supervisor 报告中暴露 brief goal/hypotheses。
- select_tools 根据 issue_type 和 brief hypotheses 优先排序。
- LLM advisor payload 包含 brief。
- 单测覆盖 health-food 推荐/配额等路径。

## Acceptance

- 相同 tools 集合下，推荐问题优先推荐状态/餐食/用户资料，配额问题优先 quota。
- Verifier 仍控制预算、scope、可用工具和 brief 绑定。
