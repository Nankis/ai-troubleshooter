# AGENTS.md

此文件是 AI Agent 进入 `ai-troubleshooter` 仓库时的入口说明。

## 工作语言

- 默认使用中文交流。
- 文档优先中文，接口名、字段名和标准术语保留英文。
- 代码、文档和 Program 改动必须保持小批次、可验证、可回滚。

## 启动顺序

1. 读取根目录 `README.md`，理解当前平台边界和部署架构。
2. 读取关键架构文档：
   - `docs/architecture-decisions.md`
   - `docs/gateway-security.md`
   - `docs/decision-logging-and-limits.md`
   - `docs/ai-connector-integration.md`
   - `docs/LESSONS.md`
3. 若用户指定 Program，读取该 Program 的 `STATUS.yml`、`PROGRAM.md`、`SCOPE.yml`、`TASKS.md` 和 `EVIDENCE.md`。
4. 若用户未指定 Program，先判断当前任务等级：
   - Tiny：查询状态、修 typo、单文件小文案，可不建 Program。
   - Lite：1-3 个文件的小修，推荐建 Program。
   - Full：跨代码、架构、接口、DDL、安全、部署或多文档调整，必须新建 Program。
5. 修改前确认：
   - 当前任务是否需要新 Program。
   - 写入范围是否被当前 Program 的 `SCOPE.yml` 或用户当前请求允许。
   - 工作树是否已有用户或其他窗口的未提交改动。
   - 是否命中 `docs/LESSONS.md` 中 `count >= 1` 的历史错误。

## 核心约束

- 不要回滚用户或其他窗口已有改动。
- 不要为了新命名、新架构或新理解去回写旧 Program 的历史上下文；独立变更必须新增 Program。
- 如果确实必须修正旧 Program 的事实错误，必须在当前新 Program 的 `DECISIONS.md` 和 `ERRORS.md` 说明原因。
- 遇到与 `docs/LESSONS.md` 反复错误计数器相似的问题时，先读取对应复盘，再设计方案；如果再次踩坑，先给对应条目 `count +1`。
- 平台数据、知识库、AI 决策日志、工具审计和 LLM/Vision provider 属于 Agent 平台边界；业务方只提供 readonly business APIs/adapters。
- Python `apps/decision-engine` 是目标 Agent Orchestrator；Go `internal/decisionbaseline` 只能作为本地 smoke/fallback。
- 完成、暂停、切换方向时，把状态写回当前 Program 文档，不依赖聊天线程保存上下文。

## Git 提交与推送

- 完成一个 Full/大需求或可验收里程碑后，默认自己完成 commit 和 push。
- 提交前至少运行 `git diff --check`；涉及 Go/Python 代码时运行 `make test`，必要时加 `go vet ./...`。
- 提交信息优先带 Program ID，说明主要变更和验证结果。
- 工作树有不属于本任务的脏改动时，只 stage 本任务相关文件。

## Program 使用口令

```text
新 Program: xxx
继续 P-YYYY-NNN
先拆 TASKS
先看 LESSONS
不要回写旧 Program
任务暂停，写 HANDOFF
任务完成，写 RESULT
错误复盘，写 ERRORS 和 docs/LESSONS.md
```
