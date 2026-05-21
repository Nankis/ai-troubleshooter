# P-2026-001 Troubleshooter Knowledge Evolution

## 背景

业务排障 Agent 不能只完成一次性报告，还必须把每次问题、工具证据、人工最终根因、反馈和可复用排障经验沉淀下来。当前仓库已有一期 MVP、工具网关和接入规范，但经验沉淀/自进化闭环还不完整，且后续开发本身需要纳入 `ai-workflow` Program 机制。

## 目标

- 补齐 root cause、case feedback、knowledge item、knowledge evolution run 的 DDL、写入、查询和自进化逻辑。
- 提供 HTTP API，让业务方可以回填根因、提交反馈、查询知识库。
- 提供 MySQL store，使部署后能持久化核心数据；本地无 DSN 时仍可用内存 store smoke。
- 补齐业务方和后续 AI 可读的知识沉淀/自进化文档。
- 将本仓库后续开发纳入 `ai-workflow` Program 机制。

## 非目标

- 不在本轮接真实 Redis Stream。
- 不在本轮接真实 Lark 发送消息 API。
- 不在本轮让 AI 自动确认 root cause；root cause 必须来自人工或可信系统。
- 不在本轮自动修改代码或工具 registry 作为自进化结果。

## 验收标准

- `migrations/002_knowledge_evolution.sql` 包含知识沉淀和自进化运行记录 DDL。
- root cause 回填能触发 knowledge item upsert 和 evolution run 记录。
- `/knowledge` 能查询沉淀后的知识条目。
- MySQL store 实现 `caseflow.Store` 的新增读写接口。
- 文档说明业务方如何回填根因、如何查询知识、后续 AI 如何维护这套闭环。
- `make test` 通过。
- 本地 dev-server smoke 跑通：创建 case、回填 root cause、查询 knowledge。

## 风险

- MySQL 环境当前本机未提供，真实 MySQL migration 执行只能记录为待业务方/测试环境验证。
- 多进程部署还需要 Redis Stream 或其它共享队列，本轮只保证一体化服务和持久化 store 具备。
