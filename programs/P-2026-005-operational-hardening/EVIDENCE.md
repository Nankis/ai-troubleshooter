# EVIDENCE

## EV-T1-PROGRAM

- 状态：PASS
- 证据：Program 文件已建立。

## EV-T2-LARK-IDEMPOTENCY

- 状态：PASS
- 证据：`caseflow.Store.FindCaseByMessageID`、memory store、MySQL store 已实现；Lark handler 对重复 `message_id` 返回 `duplicate=true`；`TestHandlerIgnoresDuplicateMessageID` 验证只 publish 一次。

## EV-T3-PROCESS-GUARD

- 状态：PASS
- 证据：Decision runner 只允许 `NEW`、`NEED_MORE_INFO`、`WAITING_USER_REPLY` 开始处理，并先认领到 `READY_TO_INVESTIGATE`；非入口状态写 `process_skipped`；陈旧 `READY_TO_INVESTIGATE` 写 `process_stale_claim_recovered` 后继续处理，陈旧 `INVESTIGATING` / `WAITING_TOOL_RESULT` 写 `process_stale_timeout` 后失败收敛。

## EV-T4-DECISION-MASKING

- 状态：PASS
- 证据：`jsonSnapshot` 写入前调用 `masking.MaskValue`；decisionbaseline 单测验证手机号和 api key 不落明文。

## EV-T5-FINAL

- 状态：PASS
- 证据：`git diff --check`、`make test`、`go vet ./...`、`go test -race ./...` 均通过。
