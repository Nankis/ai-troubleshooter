# TASKS

## Task 1: [x] 创建 Program 并确认安全边界

- debug-only。
- service_name -> repo_path 只来自本地 registry。
- Gateway 只提供 `service_name/repo_hint/suspect_area`。
- Evidence：`EV-T1-001`

## Task 2: [x] 实现本地仓库 registry 和只读 inspector

- 支持 env JSON registry。
- 支持 allowed/deny globs。
- 返回相对路径、命中词、行号，不返回源码片段。
- Evidence：`EV-T2-001`

## Task 3: [x] 接入 Python Local Code Agent

- Gateway 证据不足且显式 debug 才触发。
- 无 mapping 时失败收敛，不乱读路径。
- Verifier 支持 `local_code_inspection` 动作。
- Evidence：`EV-T3-001`

## Task 4: [x] 补正负向测试和文档

- 正向命中 allowlist 文件。
- 敏感文件 deny。
- 无 mapping。
- debug 未满足时不触发。
- Evidence：`EV-T4-001`

## Task 5: [x] 验证、提交和推送

- `make test`
- `git diff --check`
- `python3.13 scripts/secret-scan.py --mode all`
- Evidence：`EV-T5-001`
