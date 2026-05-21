# ai-workflow 开发规范接入

本仓库后续开发按 `/Users/ginseng/Documents/AI工作区/ai-workflow` 的自动化开发闭环执行。

## 本仓库执行约定

进入较大开发任务时：

1. 读取 `ai-workflow/AGENTS.md`。
2. 读取 `ai-workflow/core/DEV-FLOW.md`。
3. 读取 `ai-workflow/core/AUTOMATED-DEVELOPMENT.md`。
4. 明确任务级别、Scope、验收标准。
5. 小步改动、每步可验证。
6. 提交前必须运行：

```bash
git status --short
git diff --check
make test
```

## 分支策略

本仓库是可部署后端仓库，默认不直接 push `main`。

推荐流程：

```text
main
  -> codex/{task-name}
  -> commit
  -> push branch
  -> PR
  -> CI / review
  -> merge
```

如果本地已在 `main` 产生 commit，不要继续 push `main`；创建工作分支指向当前提交，让用户决定合并。

## DDL 变更规则

涉及 DDL 必须同步：

- `migrations/*.sql`
- Go model
- Store interface / implementation
- API handler
- docs
- unit test
- smoke test

不能只写 DDL 不写读写路径，也不能只写代码不写 migration。

## 完成定义

一次平台能力开发只有同时满足以下条件才算完成：

- 代码已实现；
- DDL 已包含；
- 插入和查询路径已包含；
- 自进化或业务逻辑已包含；
- 文档告诉业务方和后续 AI 如何接入；
- `make test` 通过；
- 本地 smoke 对核心接口跑通；
- commit 在工作分支上。
