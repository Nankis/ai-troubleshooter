# Errors

## E1: `local_rules` 被披露后仍允许排障

- 现象：关闭本地 Agent 后，Web Chat 仍然用规则、平台经验或 Gateway 证据回复用户。
- 用户影响：用户会误以为问题由真实 Agent/模型排查过，属于可信边界错误。
- 根因：P-2026-049 只做了“披露 mock/local_rules”，没有把“无真实决策 Agent 禁止排障”写成代码守门。
- 防复发：
  - 默认 `local_rules` 只能做 intake，不允许做诊断结论。
  - 守门失败必须有单测证明未调用 Gateway、未查询平台经验、未产出 orchestrator plan。
  - Program 验证不能把披露当作安全边界。

