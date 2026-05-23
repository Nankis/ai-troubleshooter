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
