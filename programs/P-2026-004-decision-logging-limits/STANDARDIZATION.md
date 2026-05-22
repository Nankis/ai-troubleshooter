# STANDARDIZATION

- 每个 AI 关键决策必须记录：case、investigation、decision type、reason、input snapshot、output snapshot、selected tools、status、latency、error。
- Decision runner 必须有 case 级 timeout、工具调用总数上限和工具失败上限。
- 失败或超时时必须把 case/investigation 收敛到终态，不能让 worker 反复重试同一个运行态 case。
