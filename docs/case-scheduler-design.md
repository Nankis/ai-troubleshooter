# Case Scheduler 设计草图

本阶段只实现最小状态机，不引入复杂 worker。目标是让排障 case 的 claim、start、finish、timeout 有统一语义，后续再平滑替换成后台队列或分布式 scheduler。

## 状态

| 状态 | 含义 |
| --- | --- |
| `NEW` | case 已创建但未进入排障 |
| `READY_TO_INVESTIGATE` | 可以被 scheduler claim |
| `INVESTIGATING` | 决策层正在排查 |
| `WAITING_TOOL_RESULT` | 正在等待 Gateway 工具 |
| `NEED_MORE_INFO` | 缺字段，需要用户补充 |
| `WAITING_USER_REPLY` | 等用户继续输入 |
| `NEED_HUMAN_CONFIRMATION` | Agent 给出有界结论，需要业务 Owner 确认 |
| `DONE` | 人工确认完成 |
| `FAILED` | 失败或超时 |

## 事件

- `scheduler_claimed`：case 被当前进程领取。
- `scheduler_rejected`：状态不允许领取。
- `scheduler_finished`：同步排障结束。
- `scheduler_failed`：排障异常。
- `scheduler_timed_out`：超过 `MAX_INVESTIGATION_SECONDS`。

## 最小实现

- Python `case_scheduler.py` 定义状态和合法迁移。
- `AgentPlatform.process_case` 在创建 supervisor run 后记录 claim 事件，在返回前记录 finish/failed 事件。
- 暂不新增 DB 表，先复用 `tb_troubleshoot_agent_run_event`，避免过早引入多 worker 复杂度。

## 后续扩展

- 多 worker claim 需要 DB 条件更新或锁。
- 长任务需要 heartbeat、stale run 回收和重试策略。
- case queue 可以从 `READY_TO_INVESTIGATE` 状态扫描，但必须保留预算、超时和幂等限制。
