from __future__ import annotations

import os
import re
from dataclasses import dataclass
from pathlib import Path


@dataclass(frozen=True, slots=True)
class MySQLConfig:
    host: str
    port: int
    user: str
    password: str
    database: str
    charset: str = "utf8mb4"


@dataclass(frozen=True, slots=True)
class LLMConfig:
    provider: str
    base_url: str
    api_key: str
    model: str
    timeout_seconds: int
    allow_rule_fallback: bool


@dataclass(frozen=True, slots=True)
class ChatPlatformConfig:
    platform: str
    api_base_url: str
    app_id: str
    app_secret: str
    verification_token: str
    encrypt_key: str
    allowed_chat_ids: tuple[str, ...]
    max_images_per_message: int
    max_image_bytes: int


@dataclass(frozen=True, slots=True)
class Config:
    host: str
    port: int
    db_driver: str
    mysql: MySQLConfig | None
    gateway_endpoint: str
    gateway_bearer_token: str
    gateway_admin_bearer_token: str
    gateway_agent_id: str
    max_tool_calls_per_case: int
    max_tool_failures_per_case: int
    max_investigation_seconds: int
    web_asset_path: Path
    llm: LLMConfig
    chat_platform: ChatPlatformConfig


def load_config() -> Config:
    repo_root = Path(__file__).resolve().parents[3]
    profile = _env("AI_MODEL_PROFILE", _env("MODEL_PROFILE", "local_rules")).lower()
    llm = _load_llm_config(profile)
    db_driver = _env("DB_DRIVER", "mysql").lower()
    mysql = _load_mysql_config() if db_driver == "mysql" else None
    return Config(
        host=_env("AGENT_PLATFORM_HOST", "127.0.0.1"),
        port=_env_int("AGENT_PLATFORM_PORT", _env_int("HTTP_PORT", 19091)),
        db_driver=db_driver,
        mysql=mysql,
        gateway_endpoint=_env("GATEWAY_ENDPOINT", _env("TOOL_GATEWAY_ENDPOINT", "http://127.0.0.1:8080")),
        gateway_bearer_token=_env("GATEWAY_BEARER_TOKEN", _env("TOOL_GATEWAY_BEARER_TOKEN", "")),
        gateway_admin_bearer_token=_env("GATEWAY_ADMIN_BEARER_TOKEN", _first_env("CONTROL_API_BEARER_TOKEN", "CONTROL_API_TOKEN")),
        gateway_agent_id=_env("GATEWAY_AGENT_ID", "business-troubleshooter-v1"),
        max_tool_calls_per_case=_env_int("MAX_TOOL_CALLS_PER_CASE", 10),
        max_tool_failures_per_case=_env_int("MAX_TOOL_FAILURES_PER_CASE", 3),
        max_investigation_seconds=_env_int("MAX_INVESTIGATION_SECONDS", 120),
        web_asset_path=Path(_env("WEB_ASSET_PATH", str(repo_root / "web" / "static" / "index.html"))),
        llm=llm,
        chat_platform=_load_chat_platform_config(),
    )


def _load_mysql_config() -> MySQLConfig:
    if _env("MYSQL_HOST", ""):
        return MySQLConfig(
            host=_env("MYSQL_HOST", "127.0.0.1"),
            port=_env_int("MYSQL_PORT", 3306),
            user=_env("MYSQL_USER", "root"),
            password=_env("MYSQL_PASSWORD", ""),
            database=_env("MYSQL_DATABASE", "ai_troubleshooter"),
        )
    dsn = _env("DB_DSN", "")
    if not dsn:
        raise RuntimeError("DB_DRIVER=mysql requires MYSQL_* env or DB_DSN")
    parsed = _parse_go_mysql_dsn(dsn)
    if parsed is None:
        raise RuntimeError("Python Agent Platform supports MYSQL_* env or Go-style DB_DSN user:pass@tcp(host:port)/database")
    return parsed


def _parse_go_mysql_dsn(dsn: str) -> MySQLConfig | None:
    match = re.match(r"([^:]+):([^@]*)@tcp\(([^:)]+)(?::(\d+))?\)/([^?]+)", dsn)
    if not match:
        return None
    user, password, host, port, database = match.groups()
    return MySQLConfig(host=host, port=int(port or "3306"), user=user, password=password, database=database)


def _load_llm_config(profile: str) -> LLMConfig:
    provider = profile
    base_url = ""
    api_key = ""
    model = "rules-v1"
    if profile in {"local", "local_rules", "rules", ""}:
        provider = "local_rules"
    elif profile in {"qwen", "dashscope"}:
        provider = "openai_compatible"
        base_url = "https://dashscope.aliyuncs.com/compatible-mode/v1"
        api_key = _first_env("DASHSCOPE_API_KEY", "QWEN_API_KEY")
        model = _env("QWEN_MODEL", "qwen-plus")
    elif profile in {"gpt", "openai"}:
        provider = "openai"
        base_url = "https://api.openai.com/v1"
        api_key = _env("OPENAI_API_KEY", "")
        model = _env("OPENAI_MODEL", "gpt-4.1-mini")
    elif profile in {"claude", "anthropic"}:
        provider = "anthropic"
        base_url = _env("ANTHROPIC_BASE_URL", "https://api.anthropic.com")
        api_key = _first_env("ANTHROPIC_API_KEY", "CLAUDE_API_KEY")
        model = _env("ANTHROPIC_MODEL", "claude-sonnet-4-5")
    elif profile == "claude_code":
        provider = "claude_code"
        base_url = _first_env("CLAUDE_CODE_BASE_URL", "ANTHROPIC_BASE_URL")
        api_key = _first_env("CLAUDE_CODE_API_KEY", "ANTHROPIC_API_KEY", "CLAUDE_API_KEY")
        model = _env("CLAUDE_CODE_MODEL", _env("ANTHROPIC_MODEL", "claude-sonnet-4-5"))
    else:
        provider = "openai_compatible"

    provider = _env("LLM_PROVIDER", provider)
    base_url = _env("LLM_BASE_URL", base_url)
    api_key = _env("LLM_API_KEY", api_key)
    model = _env("LLM_MODEL", model)
    return LLMConfig(
        provider=provider,
        base_url=base_url,
        api_key=api_key,
        model=model,
        timeout_seconds=_env_int("LLM_TIMEOUT_SECONDS", 30),
        allow_rule_fallback=_env_bool("LLM_ALLOW_RULE_FALLBACK", False),
    )


def _load_chat_platform_config() -> ChatPlatformConfig:
    platform = _normalize_chat_platform(_env("LARK_PLATFORM", "lark"))
    explicit_base = _env("LARK_API_BASE_URL", "")
    return ChatPlatformConfig(
        platform=platform,
        api_base_url=explicit_base or _default_chat_platform_base_url(platform),
        app_id=_env("LARK_APP_ID", ""),
        app_secret=_env("LARK_APP_SECRET", ""),
        verification_token=_env("LARK_VERIFICATION_TOKEN", ""),
        encrypt_key=_env("LARK_ENCRYPT_KEY", ""),
        allowed_chat_ids=tuple(_env_csv("LARK_ALLOWED_CHAT_IDS")),
        max_images_per_message=_env_int("VISION_MAX_IMAGES_PER_MESSAGE", 3),
        max_image_bytes=_env_int("VISION_MAX_IMAGE_BYTES", 10 * 1024 * 1024),
    )


def _normalize_chat_platform(value: str) -> str:
    normalized = value.strip().lower()
    if normalized in {"feishu", "feishu_cn", "cn", "china"}:
        return "feishu"
    return "lark"


def _default_chat_platform_base_url(platform: str) -> str:
    if platform == "feishu":
        return "https://open.feishu.cn"
    return "https://open.larksuite.com"


def _env(key: str, default: str) -> str:
    value = os.environ.get(key, "").strip()
    return value if value else default


def _first_env(*keys: str) -> str:
    for key in keys:
        value = os.environ.get(key, "").strip()
        if value:
            return value
    return ""


def _env_int(key: str, default: int) -> int:
    raw = os.environ.get(key, "").strip()
    if not raw:
        return default
    try:
        return int(raw)
    except ValueError:
        return default


def _env_bool(key: str, default: bool) -> bool:
    raw = os.environ.get(key, "").strip().lower()
    if not raw:
        return default
    return raw in {"1", "true", "yes", "y", "on", "enabled"}


def _env_csv(key: str) -> list[str]:
    raw = os.environ.get(key, "")
    return [item.strip() for item in raw.split(",") if item.strip()]
