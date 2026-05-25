# Handoff

## Current Goal

固化 `apply_patch`/文件写入必须绑定目标仓库根目录的操作纪律。

## Current State

- 已完成。
- `AGENTS.md` 新增“编辑路径铁律”。
- `docs/LESSONS.md` 新增 `wrong-repo-apply-patch` 计数和 2026-05-25 复盘。

## Evidence

- `git diff --check`：pass。
- `make secret-scan`：pass。
- `python3.13 scripts/validate_program.py programs/P-2026-057-repo-root-edit-discipline`：pass。

## Next Steps

- 后续任何写入前先确认 `pwd` 和 `git rev-parse --show-toplevel`。
- `apply_patch` 优先使用 `/Users/ginseng/Documents/AI工作区/ai-troubleshooter/...` 绝对路径。

## Risks

- 如果只依赖聊天记忆，这个问题还会复发；因此规则已经写入 `AGENTS.md` 和 `docs/LESSONS.md`。
