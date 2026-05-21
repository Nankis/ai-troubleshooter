# TASKS

## Program

- Program：`P-2026-001-troubleshooter-knowledge-evolution`
- 当前任务：完成

## 依赖关系总览

```text
Task 1: 建立 Program 与 Scope
  -> Task 2: 补 DDL、模型、store、自进化逻辑
  -> Task 3: 补 API 与文档
  -> Task 4: 验证、Evidence、提交推送工作分支
```

## 任务列表

### Task 1: [x] 建立 Program 与 Scope

- 文件：`programs/P-2026-001-troubleshooter-knowledge-evolution/*`
- 验收标准：
  - [x] Program、Scope、Tasks 已存在。
  - [x] 明确不直接 push main。
- Evidence ID：`EV-T1-PROGRAM`

### Task 2: [x] 补经验沉淀与自进化核心实现

- 文件：`internal/caseflow/*`、`internal/evolution/*`、`internal/storage/*`、`migrations/*`
- 验收标准：
  - [x] root cause、feedback、knowledge item、evolution run 有 Go model。
  - [x] 内存 store 支持写入/查询。
  - [x] MySQL store 支持写入/查询。
  - [x] root cause 回填触发 knowledge item upsert。
- Evidence ID：`EV-T2-TEST`

### Task 3: [x] 补 API 与文档

- 文件：`cmd/dev-server/main.go`、`docs/*`、`api/openapi/*`、`README.md`
- 验收标准：
  - [x] `/cases/{case_no}/root-cause` 可写入并触发演进。
  - [x] `/knowledge` 可查询知识条目。
  - [x] 文档说明 DDL、写入、查询、自进化规则。
  - [x] 文档说明后续开发使用 Program 机制。
- Evidence ID：`EV-T3-SMOKE`

### Task 4: [x] 验证、Evidence、提交推送工作分支

- 文件：`programs/P-2026-001-troubleshooter-knowledge-evolution/*`
- 验收标准：
  - [x] `git diff --check` 通过。
  - [x] `make test` 通过。
  - [x] dev-server smoke 通过。
  - [x] Evidence / Result / Handoff 更新。
  - [x] commit 并 push `codex/knowledge-evolution`。
- Evidence ID：`EV-T4-FINAL`
