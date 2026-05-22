# RISKS

- Python decision-engine 目前仍是 skeleton，worker 生产调用链尚未真正切过去。
- Go decisionbaseline 仍存在，后续文档和部署说明要持续避免把它误认为目标生产决策层。
- 如果后续修改旧 Program，会破坏执行记录的时间上下文，需要通过新增 Program 记录新决策。
- 如果执行者没有先读 `AGENTS.md` 和 `docs/LESSONS.md`，仍可能重复回写旧 Program 或漏建新 Program。
