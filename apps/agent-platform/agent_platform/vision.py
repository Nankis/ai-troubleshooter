from __future__ import annotations

import base64
import json
import re
import urllib.error
import urllib.request
from dataclasses import dataclass
from typing import Any

from .config import LLMConfig, VisionConfig
from .llm import _loads_json_object


@dataclass(frozen=True, slots=True)
class ImageInput:
    filename: str
    media_type: str
    data: bytes


@dataclass(frozen=True, slots=True)
class VisionResult:
    ocr_text: str = ""
    summary: str = ""
    provider: str = "local_rules"
    model: str = "local-vision-placeholder"
    is_real: bool = False


class LocalVisionClient:
    @property
    def is_real(self) -> bool:
        return False

    def analyze(self, text: str, images: list[ImageInput]) -> VisionResult:
        if not images:
            return VisionResult()
        names = ", ".join(item.filename or f"image-{idx + 1}" for idx, item in enumerate(images))
        return VisionResult(
            ocr_text=f"已收到 {len(images)} 张图片：{names}。当前未配置真实 Vision provider，仅作为附件证据保存。",
            summary="local vision placeholder",
            provider="local_rules",
            model="local-vision-placeholder",
            is_real=False,
        )


class OpenAICompatibleVisionClient:
    def __init__(self, config: VisionConfig) -> None:
        self.config = config

    @property
    def is_real(self) -> bool:
        return True

    def analyze(self, text: str, images: list[ImageInput]) -> VisionResult:
        if not images:
            return VisionResult(provider=self.config.provider, model=self.config.model, is_real=True)
        self._require_config()
        body = {
            "model": self.config.model,
            "temperature": 0.1,
            "messages": [
                {
                    "role": "system",
                    "content": (
                        "你是生产排障平台的截图识别助手。只提取图片和用户文本中的客观信息，不要编造。"
                        "只输出 JSON，字段：ocr_text, summary, key_fields, uncertainties。"
                    ),
                },
                {"role": "user", "content": self._content(text, images)},
            ],
        }
        data = self._post_json(_chat_completions_url(self.config.base_url), body, {"Authorization": f"Bearer {self.config.api_key}"})
        content = data.get("choices", [{}])[0].get("message", {}).get("content", "{}")
        payload = _loads_json_or_text(content)
        ocr_text = _string_field(payload, "ocr_text") or _string_field(payload, "text") or str(content).strip()
        summary = _string_field(payload, "summary") or _clip(ocr_text, 240)
        key_fields = payload.get("key_fields")
        uncertainties = payload.get("uncertainties")
        detail_parts = [ocr_text]
        if key_fields:
            detail_parts.append("关键字段：" + json.dumps(key_fields, ensure_ascii=False))
        if uncertainties:
            detail_parts.append("不确定内容：" + json.dumps(uncertainties, ensure_ascii=False))
        return VisionResult(
            ocr_text="\n".join(part for part in detail_parts if part),
            summary=summary,
            provider=self.config.provider,
            model=self.config.model,
            is_real=True,
        )

    def _content(self, text: str, images: list[ImageInput]) -> list[dict[str, Any]]:
        content: list[dict[str, Any]] = [
            {
                "type": "text",
                "text": (
                    "请识别这些排障截图。输出：1. 所有可读文字/OCR；2. 用户/uid、服务名、错误码、时间、页面状态等关键字段；"
                    "3. 能确定的客观现象；4. 不确定或看不清的内容。用户补充文本：\n"
                    + text
                ),
            }
        ]
        for image in images[: self.config.max_images_per_message]:
            media_type = image.media_type or "application/octet-stream"
            content.append(
                {
                    "type": "image_url",
                    "image_url": {
                        "url": "data:" + media_type + ";base64," + base64.b64encode(image.data).decode("ascii"),
                        "detail": "auto",
                    },
                }
            )
        return content

    def _post_json(self, url: str, body: dict[str, Any], headers: dict[str, str]) -> dict[str, Any]:
        request = urllib.request.Request(
            url,
            data=json.dumps(body, ensure_ascii=False).encode("utf-8"),
            method="POST",
            headers={"Content-Type": "application/json", **headers},
        )
        try:
            with urllib.request.urlopen(request, timeout=self.config.timeout_seconds) as response:
                raw = response.read().decode("utf-8")
                return json.loads(raw or "{}")
        except urllib.error.HTTPError as exc:
            raw = exc.read().decode("utf-8", errors="replace")
            raise RuntimeError(_api_error_message(exc.code, raw)) from exc

    def _require_config(self) -> None:
        missing = []
        if not self.config.base_url:
            missing.append("VISION_BASE_URL")
        if not self.config.api_key:
            missing.append("VISION_API_KEY or provider API key")
        if not self.config.model:
            missing.append("VISION_MODEL")
        if missing:
            raise RuntimeError("missing real Vision config: " + ", ".join(missing))


def build_vision_client(vision: VisionConfig, llm: LLMConfig) -> LocalVisionClient | OpenAICompatibleVisionClient:
    provider = vision.provider.lower().strip()
    if provider in {"", "local", "local_rules", "rules", "off", "disabled", "none"}:
        return LocalVisionClient()
    if provider in {"same_as_llm", "llm", "main_llm"} and llm.provider in {"", "local", "local_rules", "rules"}:
        return LocalVisionClient()
    if provider in {
        "same_as_llm",
        "llm",
        "main_llm",
        "openai",
        "gpt",
        "openai_compatible",
        "openai-compatible",
        "qwen",
        "dashscope",
        "qwen_vl",
        "qwen-vl",
        "qwen_openai_compatible",
        "qwen-openai-compatible",
        "llm_gateway",
    }:
        return OpenAICompatibleVisionClient(vision)
    raise RuntimeError(f"unsupported Vision provider {vision.provider!r}")


def _chat_completions_url(base_url: str) -> str:
    base_url = base_url.rstrip("/")
    if base_url.endswith("/chat/completions"):
        return base_url
    return base_url + "/chat/completions"


def _loads_json_or_text(text: str) -> dict[str, Any]:
    try:
        return _loads_json_object(text)
    except Exception:
        return {"ocr_text": text}


def _string_field(payload: dict[str, Any], key: str) -> str:
    value = payload.get(key)
    if isinstance(value, str):
        return value.strip()
    if value is None:
        return ""
    return json.dumps(value, ensure_ascii=False)


def _api_error_message(status: int, raw: str) -> str:
    try:
        payload = json.loads(raw or "{}")
        message = payload.get("error", {}).get("message") or payload.get("message") or raw
    except json.JSONDecodeError:
        message = raw
    message = re.sub(r"(?i)(api[_-]?key|token|authorization)[^,}\\n]*", r"\1=<redacted>", str(message))
    return f"vision api status={status} error={message}"


def _clip(value: str, limit: int) -> str:
    value = " ".join(value.split())
    if len(value) <= limit:
        return value
    return value[: limit - 3] + "..."
