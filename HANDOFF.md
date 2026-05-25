# Handoff Index

当前活跃 Program：

- 无。P-2026-052 到 P-2026-056 已完成，等待统一提交和推送 main。

当前状态：

- P-2026-052：已建立 Brief 驱动排障最小工作流契约、workflow task schema、Program 验证脚本和 case scheduler 设计草图。
- P-2026-053：`InvestigationBrief` 已进入 Python DecisionRequest、Agent Platform Context Ledger、Case API 和 Web 右侧面板。
- P-2026-054：工具计划必须绑定 `hypothesis_id`、`reason`、`expected_evidence`，Verifier 会校验并把绑定信息写入工具调用决策日志。
- P-2026-055：Decision Engine 使用 Brief 和 issue type 聚焦工具排序，health-food 推荐问题优先查推荐状态、餐食、用户资料、日志和相似案例。
- P-2026-056：最小 case scheduler 状态机已接入 `process_case`，Agent Run/Event 记录 `scheduler_claimed` 和 `scheduler_finished`。
- 真实链路已跑通：real health-food readonly adapter、Go Investigation Gateway、Python Agent Platform、MySQL `ai_troubleshooter`、本地 Codex 决策 Agent、Web UI。
- 真实 case 证据：`case_20260525_000068` 查到 health-food 用户/推荐数据，5 个 Gateway readonly tools 全成功；`case_20260525_000069` 查不到用户/推荐数据，5 个 Gateway readonly tools 全成功并返回缺失证据。
- Web 截图证据：`programs/P-2026-056-case-scheduler-state-machine/artifacts/web-case-000068-brief.png`。
- 最终验证已通过：`make test`、`make secret-scan`、`git diff --check`、`python3.13 scripts/validate_program.py programs/P-2026-052... programs/P-2026-056...`。

接手规则：

- 先读 `AGENTS.md`、`programs/README.md`，再读相关 Program 的 `HANDOFF.md`。
- Program 暂停、完成里程碑、切换方向或上下文压缩前，必须更新对应 Program 的 `HANDOFF.md` 和本文件。
- 不允许把 mock、memory 或规则兜底当作最终验收；完整验收必须使用真实 MySQL、真实 HTTP 调用和真实决策 Agent。
