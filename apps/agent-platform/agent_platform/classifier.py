from __future__ import annotations

import re
from datetime import datetime
from zoneinfo import ZoneInfo


def classify_and_extract_rules(text: str) -> dict[str, object]:
    normalized = text.lower()
    entities: dict[str, str] = {}
    uid = _extract_uid(text)
    if uid:
        entities["uid"] = uid
        entities["user_id"] = uid

    if _contains_any(normalized, "health-food", "健康", "餐", "饮食", "推荐", "token", "配额", "热量"):
        domain = "health_food"
        entities["service_name"] = "health-food"
        issue_type = _health_food_issue_type(normalized)
        if issue_type:
            entities["issue_type"] = issue_type
        if _contains_any(normalized, "今日", "今天", "today"):
            entities["recommendation_date"] = datetime.now(ZoneInfo("Asia/Shanghai")).strftime("%Y-%m-%d")
        return {"issue_domain": domain, "issue_type": issue_type, "confidence": 0.82, "entities": entities}

    if _contains_any(normalized, "k线", "kline", "行情", "价格", "成交量"):
        symbol = _first_match(text, r"\b([A-Z]{2,12}USDT)\b")
        interval = _first_match(normalized, r"\b(1m|3m|5m|15m|30m|1h|4h|1d)\b")
        if symbol:
            entities["symbol"] = symbol.upper()
        if interval:
            entities["interval"] = interval
        entities["issue_type"] = "price_mismatch" if _contains_any(normalized, "不准", "不一致", "错误") else "kline_abnormal"
        return {"issue_domain": "kline", "issue_type": entities["issue_type"], "confidence": 0.78, "entities": entities}

    if _contains_any(normalized, "余额", "资产", "冻结", "充值", "提现", "划转"):
        asset = _first_match(text, r"\b(USDT|BTC|ETH|BNB|SOL|USDC)\b")
        if asset:
            entities["asset_symbol"] = asset.upper()
        entities["issue_type"] = "asset_balance_abnormal"
        return {"issue_domain": "asset", "issue_type": "asset_balance_abnormal", "confidence": 0.76, "entities": entities}

    return {"issue_domain": "", "issue_type": "", "confidence": 0.2, "entities": entities}


def merge_llm_result(rule_result: dict[str, object], llm_payload: dict[str, object]) -> dict[str, object]:
    if not llm_payload:
        return rule_result
    merged = dict(rule_result)
    if isinstance(llm_payload.get("issue_domain"), str) and llm_payload["issue_domain"]:
        merged["issue_domain"] = llm_payload["issue_domain"]
    if isinstance(llm_payload.get("issue_type"), str) and llm_payload["issue_type"]:
        merged["issue_type"] = llm_payload["issue_type"]
    if isinstance(llm_payload.get("confidence"), int | float):
        merged["confidence"] = float(llm_payload["confidence"])
    entities = dict(rule_result.get("entities") or {})
    raw_entities = llm_payload.get("entities")
    if isinstance(raw_entities, dict):
        for key, value in raw_entities.items():
            if value is not None and str(value).strip():
                entities[str(key)] = str(value)
    if merged.get("issue_type") and "issue_type" not in entities:
        entities["issue_type"] = str(merged["issue_type"])
    merged["entities"] = entities
    return merged


def _health_food_issue_type(text: str) -> str:
    if _contains_any(text, "token", "配额", "次数", "额度", "消耗"):
        return "AI配额异常"
    if _contains_any(text, "没有每日推荐", "无每日推荐", "没有推荐", "缺少推荐", "未生成"):
        return "每日推荐缺失"
    if _contains_any(text, "推荐不准", "推荐数据不准", "健康目标", "热量不对"):
        return "每日推荐不准"
    if _contains_any(text, "餐食", "meal", "饮食记录"):
        return "餐食数据异常"
    return "health_food_abnormal"


def _extract_uid(text: str) -> str:
    patterns = [
        r"(?i)\b(uid|user_id|用户|用户id)[:：\s]*([A-Za-z0-9_\-]+)",
        r"(?i)\buid[:：\s]*([A-Za-z0-9_\-]+)",
    ]
    for pattern in patterns:
        match = re.search(pattern, text)
        if match:
            return match.group(match.lastindex or 1)
    return ""


def _first_match(text: str, pattern: str) -> str:
    match = re.search(pattern, text)
    return match.group(1) if match else ""


def _contains_any(text: str, *needles: str) -> bool:
    return any(needle.lower() in text for needle in needles)
