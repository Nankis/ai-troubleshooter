# EVIDENCE

## EV-T1-PROGRAM

- 状态：PASS
- 证据：Program 文件已建立。

## EV-T2-AUTH

- 状态：PASS
- 证据：`internal/gateway/security_test.go` 覆盖未带 token 返回 401、agent_id mismatch 返回 403、认证请求返回 200。

## EV-T3-RATELIMIT

- 状态：PASS
- 证据：`internal/gateway/security_test.go` 覆盖超过 agent QPS 返回 429；`internal/ratelimit/fixed_window_test.go` 覆盖固定窗口行为。

## EV-T4-FINAL

- 状态：PASS
- 证据：`git diff --check` 通过；`make test` 通过。
