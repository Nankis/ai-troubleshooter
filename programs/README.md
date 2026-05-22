# Programs

这里记录 `ai-troubleshooter` 仓库中的中大型 AI 协作任务。

每个 Program 是一个可恢复的任务实例，适合这些场景：

- 跨代码、架构图、接口规范、DDL、安全和部署文档的调整。
- Gateway、Decision Engine、Lark/飞书入口、MySQL schema 或 connector 契约变更。
- 需要跨窗口、跨上下文继续的任务。
- 用户指出设计错误、边界错误或流程错误，需要留下复盘和防复发规则。

推荐结构：

```text
programs/P-YYYY-NNN-slug/
├── PROGRAM.md
├── STATUS.yml
├── SCOPE.yml
├── TASKS.md
├── EVIDENCE.md
├── DECISIONS.md
├── RISKS.md
├── HANDOFF.md
├── ERRORS.md
└── RESULT.md
```

Tiny 任务可以不建 Program，但只要涉及架构、安全、接口、DDL、多文档同步或错误复盘，就应该新建 Program。

## 会话治理

Program 是长任务的恢复入口，不是聊天线程的附件。

- 每个 Program 暂停、完成里程碑或切换方向时，必须更新 `HANDOFF.md`。
- 每次完成可验证小步后，把检查命令、CI run、截图路径或人工确认写入 `EVIDENCE.md`。
- Full 级 Program 的验证结果必须按 `docs/VERIFICATION.md` 记录 Evidence 索引、命令验证、覆盖映射、未验证项和已知噪音。
- 大文件、大日志、长 JSON、完整 payload 和敏感信息只写路径和摘要，不粘贴进聊天或 Program。

## 历史记录规则

- 旧 Program 保留当时上下文，不为新命名、新架构或新理解反复回写。
- 新需求、新架构调整、用户指出的新错误，都新增 Program。
- 如果必须修正旧 Program 的事实错误，当前 Program 必须记录为什么例外、改了哪些旧记录、如何避免滥用。

## 验证结果规则

- `EVIDENCE.md` 不只写 PASS 段落；应包含索引表、命令表、覆盖映射和未验证项。
- `RESULT.md` 必须包含验证摘要、验收覆盖、commit 和残留风险。
- 文档类变更至少记录 `git diff --check`。
- 代码类变更必须记录对应测试命令和 CI run。
