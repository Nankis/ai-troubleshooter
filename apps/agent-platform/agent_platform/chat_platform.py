from __future__ import annotations

import base64
import hashlib
import hmac
import json
import time
import urllib.error
import urllib.parse
import urllib.request
from dataclasses import dataclass
from threading import Lock
from typing import Any

from cryptography.hazmat.primitives.ciphers import Cipher, algorithms, modes

from .config import ChatPlatformConfig
from .vision import ImageInput


@dataclass(frozen=True, slots=True)
class ChatEvent:
    challenge: str = ""
    token: str = ""
    chat_id: str = ""
    thread_id: str = ""
    message_id: str = ""
    reporter_user_id: str = ""
    text: str = ""
    image_keys: tuple[str, ...] = ()
    ocr_text: str = ""


class ChatPlatformError(ValueError):
    def __init__(self, message: str, status_code: int = 400) -> None:
        super().__init__(message)
        self.status_code = status_code


def parse_chat_event(raw_body: bytes, config: ChatPlatformConfig) -> ChatEvent:
    payload = _decode_event_payload(raw_body, config.encrypt_key)
    event = _extract_event(payload)
    _verify_token(event.token, config.verification_token)
    if event.challenge:
        return event
    _verify_allowed_chat(event.chat_id, config.allowed_chat_ids)
    return event


def _decode_event_payload(raw_body: bytes, encrypt_key: str) -> dict[str, Any]:
    if len(raw_body) > 1024 * 1024:
        raise ChatPlatformError("chat platform event body exceeds 1MiB")
    try:
        envelope = json.loads(raw_body.decode("utf-8") or "{}")
    except json.JSONDecodeError as exc:
        raise ChatPlatformError(f"invalid chat platform json: {exc}") from exc
    if not isinstance(envelope, dict):
        raise ChatPlatformError("chat platform payload must be a json object")
    encrypted = str(envelope.get("encrypt") or "")
    if not encrypted:
        if encrypt_key:
            raise ChatPlatformError("LARK_ENCRYPT_KEY is configured but event body is not encrypted")
        return envelope
    if not encrypt_key:
        raise ChatPlatformError("encrypted chat platform event received but LARK_ENCRYPT_KEY is not configured")
    plaintext = _decrypt_event(encrypt_key, encrypted)
    try:
        payload = json.loads(plaintext.decode("utf-8") or "{}")
    except json.JSONDecodeError as exc:
        raise ChatPlatformError(f"decrypted chat platform json is invalid: {exc}") from exc
    if not isinstance(payload, dict):
        raise ChatPlatformError("decrypted chat platform payload must be a json object")
    return payload


def _decrypt_event(encrypt_key: str, encrypted: str) -> bytes:
    try:
        ciphertext = base64.b64decode(encrypted)
    except Exception as exc:
        raise ChatPlatformError(f"decode encrypted event failed: {exc}") from exc
    if not ciphertext or len(ciphertext) % 16 != 0:
        raise ChatPlatformError("invalid encrypted event block size")
    key = hashlib.sha256(encrypt_key.encode("utf-8")).digest()
    decryptor = Cipher(algorithms.AES(key), modes.CBC(key[:16])).decryptor()
    padded = decryptor.update(ciphertext) + decryptor.finalize()
    return _pkcs7_unpad(padded, 16)


def _pkcs7_unpad(value: bytes, block_size: int) -> bytes:
    if not value or len(value) % block_size != 0:
        raise ChatPlatformError("invalid encrypted event padding length")
    padding = value[-1]
    if padding <= 0 or padding > block_size or padding > len(value):
        raise ChatPlatformError("invalid encrypted event padding")
    if value[-padding:] != bytes([padding]) * padding:
        raise ChatPlatformError("invalid encrypted event padding")
    return value[:-padding]


def _verify_token(actual: str, expected: str) -> None:
    if not expected:
        return
    if not hmac.compare_digest(actual or "", expected):
        raise ChatPlatformError("invalid lark/feishu verification token", status_code=401)


def _verify_allowed_chat(chat_id: str, allowed_chat_ids: tuple[str, ...]) -> None:
    if not allowed_chat_ids:
        return
    if not chat_id or chat_id not in allowed_chat_ids:
        raise ChatPlatformError("chat is not allowed", status_code=403)


def _extract_event(payload: dict[str, Any]) -> ChatEvent:
    if payload.get("challenge"):
        return ChatEvent(challenge=str(payload.get("challenge") or ""), token=str(payload.get("token") or ""))

    if payload.get("chat_id") or payload.get("text") or payload.get("image_keys"):
        return ChatEvent(
            token=str(payload.get("token") or ""),
            chat_id=str(payload.get("chat_id") or ""),
            thread_id=str(payload.get("thread_id") or ""),
            message_id=str(payload.get("message_id") or ""),
            reporter_user_id=str(payload.get("user_id") or payload.get("reporter_user_id") or ""),
            text=str(payload.get("text") or ""),
            image_keys=tuple(_bounded_image_keys(payload.get("image_keys") or [])),
            ocr_text=str(payload.get("ocr_text") or ""),
        )

    header = payload.get("header") if isinstance(payload.get("header"), dict) else {}
    event = payload.get("event") if isinstance(payload.get("event"), dict) else {}
    message = event.get("message") if isinstance(event.get("message"), dict) else {}
    sender = event.get("sender") if isinstance(event.get("sender"), dict) else {}
    sender_id = sender.get("sender_id") if isinstance(sender.get("sender_id"), dict) else {}
    content = str(message.get("content") or "")
    extracted = ChatEvent(
        token=str(header.get("token") or payload.get("token") or ""),
        chat_id=str(message.get("chat_id") or ""),
        thread_id=str(message.get("thread_id") or message.get("root_id") or ""),
        message_id=str(message.get("message_id") or header.get("event_id") or ""),
        reporter_user_id=str(sender_id.get("open_id") or sender_id.get("user_id") or ""),
        text=_extract_text(content),
        image_keys=tuple(_bounded_image_keys(_extract_image_keys(content))),
    )
    if not extracted.chat_id and not extracted.message_id and not extracted.text and not extracted.image_keys:
        raise ChatPlatformError("unsupported lark/feishu event payload")
    return extracted


def _extract_text(content: str) -> str:
    content = content.strip()
    if not content:
        return ""
    decoded = _loads_json_or_none(content)
    if isinstance(decoded, dict):
        text = decoded.get("text")
        if isinstance(text, str):
            return text
    return content


def _extract_image_keys(content: str) -> list[str]:
    decoded = _loads_json_or_none(content)
    if decoded is None:
        return []
    keys: list[str] = []

    def walk(value: Any) -> None:
        if isinstance(value, dict):
            for key, child in value.items():
                lowered = str(key).lower()
                if lowered in {"image_key", "imagekey"} and isinstance(child, str):
                    keys.append(child)
                else:
                    walk(child)
        elif isinstance(value, list):
            for child in value:
                walk(child)
        elif isinstance(value, str) and value.startswith("img_"):
            keys.append(value)

    walk(decoded)
    return keys


def _loads_json_or_none(value: str) -> Any:
    try:
        return json.loads(value)
    except json.JSONDecodeError:
        return None


def _bounded_image_keys(values: Any, max_images: int = 20) -> list[str]:
    out: list[str] = []
    seen: set[str] = set()
    iterable = values if isinstance(values, (list, tuple)) else []
    for raw in iterable:
        value = str(raw).strip()
        if not value or value in seen:
            continue
        seen.add(value)
        out.append(value)
        if len(out) >= max_images:
            break
    return out


class LarkImageDownloader:
    def __init__(self, config: ChatPlatformConfig) -> None:
        self._config = config
        self._lock = Lock()
        self._token = ""
        self._token_expiry = 0.0

    def enabled(self) -> bool:
        return bool(self._config.app_id and self._config.app_secret)

    def download_images(self, event: ChatEvent) -> tuple[list[ImageInput], list[str]]:
        keys = _bounded_image_keys(list(event.image_keys), self._config.max_images_per_message)
        if not keys:
            return [], []
        if not self.enabled():
            return [], ["图片未下载：未配置 LARK_APP_ID/LARK_APP_SECRET；image_keys=" + ",".join(keys)]
        images: list[ImageInput] = []
        notes: list[str] = []
        for image_key in keys:
            try:
                images.append(self._download_image(event.message_id, image_key))
            except Exception as exc:
                notes.append(f"图片下载失败 image_key={image_key} error={exc}")
        return images, notes

    def _download_image(self, message_id: str, image_key: str) -> ImageInput:
        if not message_id:
            raise ChatPlatformError("message_id is required to download image")
        token = self._tenant_access_token()
        path = (
            "/open-apis/im/v1/messages/"
            + urllib.parse.quote(message_id, safe="")
            + "/resources/"
            + urllib.parse.quote(image_key, safe="")
        )
        request = urllib.request.Request(self._config.api_base_url.rstrip("/") + path, method="GET", headers={"Authorization": f"Bearer {token}"})
        try:
            with urllib.request.urlopen(request, timeout=10) as response:
                data = response.read(self._config.max_image_bytes + 1)
                media_type = response.headers.get("Content-Type") or "application/octet-stream"
        except urllib.error.HTTPError as exc:
            body = exc.read(512).decode("utf-8", "replace")
            raise RuntimeError(f"lark image download status={exc.code} body={body}") from exc
        if len(data) > self._config.max_image_bytes:
            raise ChatPlatformError(f"image exceeds max {self._config.max_image_bytes} bytes")
        return ImageInput(filename=image_key, media_type=media_type, data=data)

    def _tenant_access_token(self) -> str:
        with self._lock:
            if self._token and time.time() < self._token_expiry:
                return self._token
        body = json.dumps({"app_id": self._config.app_id, "app_secret": self._config.app_secret}).encode("utf-8")
        request = urllib.request.Request(
            self._config.api_base_url.rstrip("/") + "/open-apis/auth/v3/tenant_access_token/internal",
            data=body,
            method="POST",
            headers={"Content-Type": "application/json; charset=utf-8"},
        )
        with urllib.request.urlopen(request, timeout=10) as response:
            payload = json.loads(response.read().decode("utf-8") or "{}")
        if int(payload.get("code") or 0) != 0 or not payload.get("tenant_access_token"):
            raise RuntimeError(f"lark tenant token failed code={payload.get('code')} msg={payload.get('msg')}")
        expire = int(payload.get("expire") or 7200)
        token = str(payload["tenant_access_token"])
        with self._lock:
            self._token = token
            self._token_expiry = time.time() + max(expire - 60, 60)
        return token
