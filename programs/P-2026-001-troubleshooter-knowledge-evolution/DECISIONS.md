# DECISIONS

## D1: 自进化以人工 root cause 作为触发器

- 决策：不让 AI 自动确认根因；AI 只生成初步判断，最终 root cause 由人工或可信系统回填。
- 原因：生产排障结论高风险，必须保留人工确认门槛。

## D2: 一期自进化只更新知识库，不自动改工具或代码

- 决策：自进化结果写入 `knowledge_items` 和 `knowledge_evolution_runs`。
- 原因：避免 Agent 未经验证修改生产工具链。

## D3: 本仓库后续开发走 Program 分支流程

- 决策：平台级开发用 `programs/` 记录 Scope、Tasks、Evidence，不直接 push main。
- 原因：该仓库将成为可部署后端平台，需要更强交付证据。
