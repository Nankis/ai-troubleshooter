# P-2026-009 Web Chat Verification Hardening

## 背景

用户追问上一轮验证是否真的跑通 Web 端、是否能多测场景，以及 Gateway 鉴权、脱敏、超时 case 是否补齐。本 Program 用于补充真实验证证据和缺失的网关安全单测。

## 目标

- 补测 Web Chat 多场景：K线完整、资产完整、缺字段追问、图片识别、浏览器界面提交。
- 补齐 Gateway 直接单测：工具输出脱敏、审计参数脱敏、handler 超时返回 504。
- 重新跑全量验证并推送。

## 非目标

- 不接真实业务下游接口。
- 不接真实 Lark/Feishu bot。
- 不提交任何模型 key、MySQL 密码或 token。
