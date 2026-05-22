# 架构决策记录

本文档记录影响系统方向的长期决策。运行代码可以后续分阶段调整，但这里先固定共识，避免实现越走越散。

## ADR-001：决策层改为 Python，Gateway 和下游适配保持稳定

日期：2026-05-22

### 背景

当前决策层主要由 Go orchestrator 实现，适合把一期闭环跑通，也方便和 Gateway、store、worker 放在一个进程里验证。但后续如果要让智能体更聪明，决策层会越来越依赖 Python 生态：

- 多模型编排，例如 Qwen-VL 识图，GPT/Claude 做文本推理。
- RAG、embedding、rerank、prompt template、eval dataset。
- LangGraph / LlamaIndex / DSPy / 自研 workflow。
- 本地 Claude Code / Cursor Agent 协助读取本地代码。
- 更快地调 prompt、调策略、做离线评测。

### 决策

决策层后续迁移为 Python 3.13 服务，负责：

- case intake 后的分类、实体抽取、必要字段判断。
- 历史经验检索和相似 case 检索。
- 有限工具计划生成。
- 调用 Gateway 的只读工具。
- 总结证据、给出疑似原因和下一步建议。
- 写入 AI 决策日志，保留为什么这样判断、为什么调用这些工具、为什么停止。

Gateway、业务 connector、安全鉴权、审计、限流、脱敏、只读工具注册继续保持稳定，不因为决策层语言切换而改变安全边界。

目标边界：

```text
Lark / Feishu / Web Chat
  -> Case API / Queue
  -> Python Decision Engine
       -> Tool Catalog / RAG / Local Code Inspector
       -> Gateway readonly tools
  -> Investigation Gateway
       -> auth / scope / rate limit / audit / masking
       -> readonly business adapters
  -> MySQL / Postgres
       -> tb_troubleshoot_* case / message / decision / audit / knowledge tables
```

### 约束

- Python 决策层不能直接访问生产 DB、Redis、日志平台或业务服务。
- Python 决策层只能通过 Gateway 调用已注册的只读工具。
- Gateway 返回给决策层的数据必须已经过权限校验、限流、timeout 和脱敏。
- 决策层可以本地运行用于开发、联调和评测；稳定后应部署到受控环境。
- 本地代码查看只是最后手段，输出应标记为 `suspected_code_bug`，不能直接当作最终根因。
- 任何决策链路都必须保留 tool call budget、model call budget、case timeout 和 decision logs。

### 迁移方式

第一阶段不推翻 Go 实现，先把 Go orchestrator 当作 baseline：

1. 定义 Python Decision Engine 的输入输出协议。
2. Go orchestrator 保留为 fallback 或 compatibility mode。
3. Python 决策层读取 Gateway `/tools` 或 tool catalog，不读取 Gateway 源码作为运行时依据。
4. 灰度时使用 `mode=mock`、`mode=staging`、`mode=prod-shadow` 三种模式。
5. 稳定后再把 case 处理入口切到 Python 决策层。

## ADR-002：首发不强依赖向量数据库，先预留 RAG 接口

日期：2026-05-22

### 背景

RAG 对排障智能体有价值，但首发就引入独立向量数据库会增加部署、数据同步、权限、清理、评测和运维复杂度。当前系统的首要目标是快速定位生产问题，而不是先构建完整知识检索平台。

早期真正决定效果的通常是：

- Gateway 工具是否覆盖关键生产证据。
- case 字段是否抽取准确。
- 工具调用是否有边界、有审计、有停止条件。
- 人工回填 root cause 是否能沉淀为结构化经验。
- 决策层是否能稳定复用历史 case 和 SOP。

### 决策

首发不强依赖向量数据库。系统先实现一个抽象的 `KnowledgeRetriever` / `RAGRetriever` 接口，底层可以先用简单方案：

- MySQL / Postgres 结构化查询。
- issue domain、issue type、root cause category、owner service 标签过滤。
- SQL LIKE / full-text search。
- 最近成功 case、失败 case、人工确认 root cause。
- 手写 SOP / runbook 文档的关键词检索。

当经验库数据量、相似问题数量和 SOP 文档规模增长后，再引入向量库作为检索后端。

### 推荐分阶段

Phase 0：无向量库

- 主库继续保存 `tb_troubleshoot_case`、`tb_troubleshoot_case_message`、`tb_troubleshoot_ai_decision_log`、`tb_troubleshoot_tool_call_audit`、`tb_troubleshoot_root_cause`、`tb_troubleshoot_knowledge_item`。
- `uid` 使用 `VARCHAR(128)`，兼容数字 UID、字符串 UID 和平台外部用户 ID；模板里的通用 `status` 保留为行状态，业务状态统一放到 `case_status`、`investigation_status`、`decision_status`、`knowledge_status` 等字段。
- 决策层通过结构化条件查历史经验。
- RAG 接口存在，但实现为 SQL/tag/keyword retriever。

Phase 1：轻量向量能力

- 如果主库切 Postgres，优先用 pgvector，减少部署组件。
- 如果继续 MySQL，向量库作为旁路索引，优先考虑 Qdrant 或公司已有搜索/向量平台。
- 向量库只存可检索文本 chunk 和 embedding，不作为 case 主库。

Phase 2：完整 RAG

- 增加 embedding pipeline、chunk version、rerank、召回评测集。
- 将知识项、历史 case、SOP、工具说明、接口说明纳入统一检索。
- 检索结果必须带来源、版本、更新时间和置信度，便于审计。

### 不做什么

- 不把向量数据库当主库。
- 不用向量库保存审计日志、状态机、工具调用记录、幂等键或权限数据。
- 不让模型只凭相似 case 给最终根因。
- 不把本地代码检索结果写成未脱敏的大段源码日志。

### 引入向量库的触发条件

满足以下任意条件，再正式引入向量库：

- `tb_troubleshoot_knowledge_item` 超过几百条，关键词检索明显召回不足。
- 历史 case 超过几千条，需要相似 case 聚类。
- SOP / runbook 文档较多，人工维护标签成本变高。
- 离线评测显示向量召回能显著提升定位准确率。
- 公司已有稳定向量平台，可以低成本接入。

结论：先把 RAG 设计成接口，不把向量库做成首发依赖。这样系统一开始更简单，后续又不会堵住智能化演进路线。
