# P-2026-028 Agent Integrity Audit

## 背景

用户指出：上一轮 Web 工作台经验录入没有实际写入本地 MySQL，暴露出 AI 在验收时用 memory/mock 快速交差、结论说得过满的问题。本轮必须系统检查当前仓库所有主要功能的证据等级，整理历史错误，更新 Agent 工作规则，避免后续继续自作聪明。

## 目标

- 从 README、AGENTS、LESSONS、Program ERRORS/EVIDENCE/RESULT 中盘点当前功能和历史错误。
- 建立证据等级矩阵，区分文档、单测、mock、本地真实依赖、生产真实接口。
- 补跑核心平台数据路径：Web Chat、AI decision log、tool audit、root cause 自进化、Web 知识 CRUD 的 MySQL-backed 验证。
- 更新 `AGENTS.md`，规则要硬但不能膨胀。
- 更新 `docs/LESSONS.md`，把已犯错误归类成计数器。
- 修正 README 中容易被误读成“真实已验收”的状态表述。

## 非目标

- 不接真实 Lark/Feishu bot。
- 不接真实生产 health-food、DMS 或公司日志平台。
- 不改旧 Program 历史记录；本 Program 只做当前审计和结论。

## 验收标准

- `FEATURE_AUDIT.md` 覆盖当前主要功能域，并给出证据等级、证据路径和结论。
- `ERROR_SUMMARY.md` 汇总历史错误和防复发动作。
- `AGENTS.md` 简洁明确地加入证据等级、mock/memory 禁止冒充真实验收、MySQL 持久化验收规则。
- `docs/LESSONS.md` 计数器覆盖本次发现的重复错误类型。
- 本地 MySQL-backed Web UI 知识预览/编辑/删除通过，并查表证明。
- Web Chat case、message、AI decision log、tool audit、root cause、knowledge evolution run 均能在 MySQL 查到。
- `make test`、`go vet ./...`、`make secret-scan`、`git diff --check` 通过。
