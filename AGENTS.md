# AGENTS.md

此文件是 AI Agent 进入 `ai-troubleshooter` 仓库时的入口说明。它只写硬约束，不写流水账。

## 工作语言

- 默认中文交流；接口名、字段名、标准协议保留英文。
- 改动保持小批次、可验证、可回滚。

## 启动顺序

1. 读 `README.md`，确认平台边界和当前实现状态。
2. 读必需规则：`docs/LESSONS.md`、`docs/VERIFICATION.md`、`programs/README.md`。
3. 涉及架构、安全、Gateway、Decision、DDL、Lark/飞书、MCP 或业务接入时，再读相关专题文档。
4. 判断任务级别：
   - Tiny：查询状态、typo、单文件小文案，可不建 Program。
   - Lite：1-3 个文件的小修，推荐建 Program。
   - Full：跨代码、架构、接口、DDL、安全、部署、多文档或错误复盘，必须新建或继续 Program。
5. 修改前检查工作树；只改本任务范围，不回滚用户或其他窗口改动。

## 验收纪律

结论不能高于证据等级：

- L0：文档/设计。
- L1：单测、schema、静态检查。
- L2：本地 mock/fake/smoke 链路。
- L3：本地真实依赖，例如 MySQL、真实本地业务服务、真实本地代码仓。
- L4：预发/生产真实接口或真实外部平台。

硬规则：

- 使用 `mock`、`fake`、`memory`、`local_rules` 时，结论必须明说证据等级，不能写成真实业务验收。
- 平台持久化验收必须用 `DB_DRIVER=mysql`、执行 migration、通过 UI/API 写入、查询 MySQL 表、重启后再次读取。只有显式 `DB_DRIVER=memory` 才能做一次性 smoke。
- UI 验收必须实际打开页面并操作；如果只用 curl/API，只能称为 API 验证。
- 业务接入验收必须写清证据来源：mock 只能证明契约和链路；真实验收必须调用真实服务/DB/日志/生产只读接口。
- Lark/飞书、LLM/Vision、DMS、MCP 等外部系统没有真实凭据或真实端点时，必须标为未验证，不能用本地替代物冒充。
- Full 级任务的 `EVIDENCE.md` 和 `RESULT.md` 必须包含索引、命令、现场证据、覆盖映射、未验证项和已知噪音。

## 历史错误

- 命中 `docs/LESSONS.md` 中 `count >= 1` 的场景时，先读复盘再动手。
- 再次踩同类坑，先给计数器 `count +1`，再写当前 Program 的 `ERRORS.md`。
- 不为新命名、新架构或新理解回写旧 Program；独立变更新增 Program。确需修正旧事实，必须在当前 Program 写明例外。

## 架构边界

- 平台数据、知识库、AI 决策日志、工具审计、LLM/Vision provider 属于 Agent 平台；业务方只提供 readonly business APIs/adapters。
- Investigation Gateway 只管业务生产证据查询边界，不查平台 MySQL。
- Python `apps/decision-engine` 是目标 Agent Orchestrator；Go `internal/decisionbaseline` 只能作为本地 smoke/fallback。

## 开发规范

- Go 代码按 Uber Go Style Guide / Go Code Review Comments 的可维护性原则执行：小函数、清晰错误处理、资源 `defer` 释放、避免可变全局和无等待 goroutine。
- Python 代码按 Google Python Style Guide 的可读性原则执行：类型边界清楚、函数职责单一、异常语义明确，不写隐式副作用脚本。
- DB 访问优先 ORM、Query Builder 或仓库层封装；保留 raw SQL 时必须满足：
  - 所有外部输入只能走占位符参数绑定，禁止 f-string、`fmt.Sprintf`、字符串拼接把变量写进 SQL 文本。
  - 动态字段、排序、表名、状态枚举只能来自代码内白名单，不能来自用户输入。
  - 新增动态 SQL 必须有单测证明注入 payload 只出现在 args，不出现在 query。
  - Python adapter 连接 MySQL 必须使用 DB-API 参数化执行；禁止用 `mysql -e` 拼接 SQL。
- 安全参考：OWASP SQL Injection Prevention、GitHub CodeQL SQL injection 规则、Uber Go Style Guide、Google Python Style Guide。规范链接写入相关 Program 或 PR 描述。

## Git

- Full/大需求或可验收里程碑完成后，默认 commit + push。
- 提交前至少跑 `git diff --check` 和 `make secret-scan`；涉及 Go/Python 跑 `make test`，必要时加 `go vet ./...`。
- 验证结果写入当前 Program，不能只写在聊天回复里。
- 只 stage 本任务文件；提交信息优先带 Program ID 或清楚说明变更与验证。
