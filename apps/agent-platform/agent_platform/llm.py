from __future__ import annotations

import json
import re
import urllib.request
from dataclasses import dataclass
from typing import Any

from .config import LLMConfig


@dataclass(slots=True)
class LLMResult:
    payload: dict[str, Any]
    provider: str
    model: str


class LLMClient:
    def __init__(self, config: LLMConfig) -> None:
        self.config = config

    @property
    def is_real(self) -> bool:
        return self.config.provider not in {"", "local", "local_rules", "rules"}

    def classify_and_extract(self, text: str) -> LLMResult:
        if not self.is_real:
            return LLMResult({}, "local_rules", "rules-v1")
        prompt = (
            "你是生产业务工单排障 Agent。请从用户问题中抽取 JSON，字段："
            "issue_domain, issue_type, confidence, entities。entities 是字符串键值对象。"
            "只输出 JSON。"
        )
        return self._complete_json(prompt, {"text": text})

    def summarize(self, case: dict[str, Any], observations: list[dict[str, Any]]) -> LLMResult:
        if not self.is_real:
            return LLMResult({}, "local_rules", "rules-v1")
        prompt = (
            "你是生产排障 Agent。根据只读工具证据给出可审计结论 JSON，字段："
            "summary, confidence。不要编造没有证据的根因。只输出 JSON。"
        )
        return self._complete_json(prompt, {"case": case, "observations": observations})

    def _complete_json(self, prompt: str, payload: dict[str, Any]) -> LLMResult:
        provider = self.config.provider.lower().strip()
        if provider in {"openai", "openai_compatible", "gpt", "qwen", "dashscope", "deepseek", "moonshot", "llm_gateway"}:
            return self._openai_compatible(prompt, payload)
        if provider in {"anthropic", "claude", "claude_code"}:
            return self._anthropic(prompt, payload)
        raise RuntimeError(f"unsupported LLM provider {self.config.provider!r}")

    def _openai_compatible(self, prompt: str, payload: dict[str, Any]) -> LLMResult:
        self._require_real_config()
        body = {
            "model": self.config.model,
            "temperature": 0.1,
            "response_format": {"type": "json_object"},
            "messages": [
                {"role": "system", "content": prompt},
                {"role": "user", "content": json.dumps(payload, ensure_ascii=False)},
            ],
        }
        data = self._post_json(self.config.base_url.rstrip("/") + "/chat/completions", body, {"Authorization": f"Bearer {self.config.api_key}"})
        content = data.get("choices", [{}])[0].get("message", {}).get("content", "{}")
        return LLMResult(_loads_json_object(content), self.config.provider, self.config.model)

    def _anthropic(self, prompt: str, payload: dict[str, Any]) -> LLMResult:
        self._require_real_config()
        base = self.config.base_url.rstrip("/")
        url = base if base.endswith("/v1/messages") else base + ("/messages" if base.endswith("/v1") else "/v1/messages")
        headers = {"x-api-key": self.config.api_key, "anthropic-version": "2023-06-01"}
        if self.config.provider == "claude_code":
            headers["Authorization"] = f"Bearer {self.config.api_key}"
        body = {
            "model": self.config.model,
            "max_tokens": 2048,
            "temperature": 0.1,
            "system": prompt,
            "messages": [{"role": "user", "content": json.dumps(payload, ensure_ascii=False)}],
        }
        data = self._post_json(url, body, headers)
        text = "\n".join(part.get("text", "") for part in data.get("content", []) if isinstance(part, dict))
        return LLMResult(_loads_json_object(text), self.config.provider, self.config.model)

    def _post_json(self, url: str, body: dict[str, Any], headers: dict[str, str]) -> dict[str, Any]:
        request = urllib.request.Request(
            url,
            data=json.dumps(body, ensure_ascii=False).encode("utf-8"),
            method="POST",
            headers={"Content-Type": "application/json", **headers},
        )
        with urllib.request.urlopen(request, timeout=self.config.timeout_seconds) as response:
            raw = response.read().decode("utf-8")
        return json.loads(raw or "{}")

    def _require_real_config(self) -> None:
        missing = []
        if not self.config.base_url:
            missing.append("LLM_BASE_URL or AI_MODEL_PROFILE")
        if not self.config.api_key:
            missing.append("LLM_API_KEY or provider API key")
        if not self.config.model:
            missing.append("LLM_MODEL")
        if missing:
            raise RuntimeError("missing real LLM config: " + ", ".join(missing))


def _loads_json_object(text: str) -> dict[str, Any]:
    cleaned = re.sub(r"^```(?:json)?\s*|\s*```$", "", text.strip(), flags=re.IGNORECASE | re.MULTILINE)
    start = cleaned.find("{")
    end = cleaned.rfind("}")
    if start >= 0 and end >= start:
        cleaned = cleaned[start : end + 1]
    value = json.loads(cleaned or "{}")
    if not isinstance(value, dict):
        raise ValueError("LLM returned non-object JSON")
    return value
