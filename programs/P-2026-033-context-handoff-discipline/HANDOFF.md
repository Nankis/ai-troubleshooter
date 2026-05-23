# Handoff

## 当前目标

固化上下文压缩/恢复时的 Program `HANDOFF.md` 规则，防止长线程压缩后丢失关键状态。

## 已完成

- `AGENTS.md` 新增“上下文交接铁律”。
- `programs/README.md` 补充压缩前、压缩后必须使用 `HANDOFF.md`。
- P-2026-032 已补 `HANDOFF.md`。
- 本 Program 已记录错误和防复发规则。

## 下一步

- 本 Program 已完成。后续压缩恢复时，先读本文件、`AGENTS.md` 和 `programs/README.md`。
- 如继续开发，确认工作树只剩非本任务的 `.idea/` 未跟踪文件。

## 工作树提示

- 未跟踪 `.idea/` 是本地 IDE 文件，不属于本 Program，不要提交。

## 风险

- 后续 Agent 若只读聊天不读 Program 文件，仍会复发；因此规则已写入 `AGENTS.md` 启动必读路径。
