# P-2026-032 Dev Standards SQL Hardening

## 目标

补齐 AI Agent 开发规范，并审计 Go/Python DB 访问，修复任何把外部输入拼进 SQL 文本的实现。

## 范围

- `AGENTS.md`
- Go MySQL repository/query builder
- Python readonly adapter SQL 访问
- 对应单测、扫描和验证记录
