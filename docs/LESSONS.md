# 开发复盘记录

这里记录已经确认的坑点、修复方式和后续规则。只写可复用经验，不写流水账。

## 反复错误计数器

规则：每次再次踩到同类坑，先把对应 `count` 加 1，再写本次现象、根因、修复和防复发规则。执行新任务时，如果问题命中 `count >= 1` 的条目，必须先读相关复盘，再制定方案和验证步骤。

| id | count | 触发词/场景 | 优先动作 |
| --- | ---: | --- | --- |
| `program-first-discipline` | 1 | 平台级/后端能力扩展先改代码，后补 Program | 先建或读取 Program，再改代码；如果已犯错，补 Program 并在 `ERRORS.md` 写清 |
| `program-history-rewrite` | 1 | 为了新架构、新命名或一致性，改写旧 Program 的历史上下文 | 不回写旧 Program；新建 Program 记录新决策；若必须修旧事实错误，要在当前 Program 写明例外原因 |
| `mock-as-real-evidence` | 1 | 用 mock/fake 结果描述业务真实接入或生产验收 | 降级结论为 L2；真实验收必须调用真实服务/DB/日志/生产只读接口 |
| `memory-as-persistence-evidence` | 1 | 用 `DB_DRIVER=memory` 验证平台数据、经验、审计或决策日志落库 | 改用 MySQL，查表并重启后读取；memory 只能写一次性 smoke |
| `browser-input-capability-gap` | 2 | 浏览器自动化输入受虚拟剪贴板或坐标限制影响 | 区分“页面交互验证”和“坐标/键盘输入验证”；必要时用 Chrome 调试端口并记录 |
| `model-output-overtrust` | 1 | 让模型输出单点决定分类、字段或排障路径 | 模型只作候选信号；关键字段必须有规则/校验 fallback |
| `adapter-contract-mismatch` | 3 | adapter 字段名、nullable 时间、历史 DDL 不符合契约 | 接入前做 schema/envelope/nullable/字段归一化测试 |
| `low-signal-code-evidence` | 1 | 本地代码辅助只返回路径/命中词/行号，开发者无法继续排查 | 代码排查结果必须包含文件、符号/方法、行范围、疑点、下一步核对建议和必要的有界脱敏摘录 |
| `local-schema-sprawl` | 1 | 本地 MySQL 为每个 Program 或 adapter 创建新的排障 schema | 统一使用 `ai_troubleshooter`；非 canonical schema 必须显式开关和清理计划 |

## 2026-05-23：不要为了新架构回写旧 Program

- 现象：在修正 Agent 平台边界和 Python 决策层归属时，为了让全仓 `orchestrator` 命名一致，把 P-2026-001、P-2026-003、P-2026-004、P-2026-005 等旧 Program 的历史表述同步改成了新命名。
- 根因：把“当前架构文档要一致”和“Program 是历史执行记录”混在了一起。Program 应记录当时的任务上下文和证据链，后续架构调整应该追加新 Program，而不是回写旧 Program。
- 修复：新增 `P-2026-006-architecture-boundary-alignment` 专门记录本次架构边界调整，并补充 `AGENTS.md`、`programs/README.md` 和本复盘文件。
- 规则：以后遇到新边界、新命名、新目录调整时，先判断是否是独立变更；只要是独立变更就新建 Program。旧 Program 只在修正事实错误时例外，并且必须在当前 Program 的 `DECISIONS.md` / `ERRORS.md` 记录原因。

## 2026-05-23：不要用低等级证据冒充高等级验收

- 现象 1：health-food 初期接入用 mock adapter 证明了链路，却容易被描述成真实业务接入；后续 P-2026-012 改为真实本地服务、真实 DB 和 readonly adapter 验证。
- 现象 2：Web 工作台经验录入曾用 `DB_DRIVER=memory` 启动，导致用户手动查 MySQL 找不到记录；P-2026-027 已改成 MySQL 缺 DSN fail-fast，并补 MySQL UI 落库和重启验证。
- 根因：没有强制“结论不能高于证据等级”，也没有把 mock/memory/local_rules 的限制写进入口规则。
- 修复：`AGENTS.md` 新增 L0-L4 证据等级和持久化验收硬规则；P-2026-028 对现有功能重新做证据矩阵。
- 规则：mock/fake/memory/local_rules 只能证明对应层级；涉及业务真实、生产、持久化、用户可见 UI 时，必须跑到对应真实依赖并记录查询/截图/API/表数据。

## 2026-05-23：浏览器输入能力限制要透明记录

- 现象：多次遇到 Browser 插件虚拟剪贴板不可用，`type/fill` 不稳定。
- 根因：把“页面打开并可读”误当成“输入/提交能力一定可用”。
- 修复：P-2026-027 和 P-2026-028 使用本机 Chrome 调试端口完成真实页面录入，同时在 Evidence 写明浏览器插件限制。
- 规则：UI 验收必须说明操作方式。若使用 DOM eval/CDP，也必须打开真实页面、触发真实前端事件、保留截图或 DOM/API 证据。

## 2026-05-25：本地代码辅助必须能指导开发者下一步

- 现象：Local Code Agent 回复只有相对路径、行号和命中词，开发者看完不知道具体方法、哪几行可疑、为什么可疑，也不知道下一步如何核对。
- 根因：分析层已有符号和调用边，但平台回复丢掉了这些结构化信息；证据结构缺少面向开发者的疑点和核对建议。
- 修复：P-2026-043 将本地代码 evidence 升级为 actionable finding，包含文件、方法/符号、行范围、可疑原因、下一步核对建议和有界脱敏代码摘录。
- 规则：以后凡是“代码辅助排查”类输出，不能只给搜索结果；必须给开发者可直接打开文件验证的定位卡片。

## 2026-05-25：本地 MySQL 不要为每个验证创建新 schema

- 现象：本地 MySQL 出现多个排障相关 schema，包括 `ai_troubleshooter_hf_codex`、`ai_troubleshooter_hf_real`、`ai_troubleshooter_itest`、`ai_troubleshooter_p2026008` 和 `hf_troubleshoot_codex`。
- 根因：早期 Program 为了隔离验证直接改 `MYSQL_DATABASE`；迁移脚本会对任意库名执行 `CREATE DATABASE IF NOT EXISTS`；health-food readonly adapter 文档和默认值也鼓励了临时库名。
- 修复：P-2026-045 将本地平台库固定为 `ai_troubleshooter`，在 migration、Agent Platform 和 Go Gateway 增加本地非 canonical schema fail-fast；health-food adapter 不再默认临时业务库；新增只审计、不默认删除的 schema audit 脚本。
- 规则：本地验证用 case/test data 区分，不用新 schema 区分。确需隔离实验时必须设置 `ALLOW_NON_CANONICAL_LOCAL_DB=true`，并在当前 Program 写明为什么隔离、什么时候清理、用什么脚本清理。
