# DECISIONS

## D1. 借鉴 Multica 的生命周期，不引入 Multica 依赖

Multica 的价值在于 managed agents 的任务生命周期和 runtime 抽象。本项目保留当前 Python Agent Platform / Python Decision Engine / Go Gateway 架构，只在平台内部引入 Agent Run 记录和 Local Runtime 契约。

## D2. Agent Run 属于平台数据

Agent Run、Run Event、Runtime 注册信息都属于 Agent 平台数据，写入平台 MySQL。业务方不需要提供这些表，也不通过 Investigation Gateway 查询这些平台数据。

## D3. Local Runtime 先只读、debug-only

后续本地 runtime 可以调用 Codex / Claude Code / Cursor Agent 查看本地代码，但默认只做只读分析。任何代码修改能力必须另起 Program 并加显式授权、隔离工作区和验证规则。

## D4. Case 仍是排障主实体

不把系统改造成 issue tracker。Multica 的 issue 映射为本项目的 troubleshoot case；agent run 是 case 的执行轨迹，不取代 case。
