from __future__ import annotations

import json
import os
import platform
import re
import shutil
import socket
import subprocess
import tempfile
from dataclasses import asdict, dataclass, field
from pathlib import Path
from typing import Any


@dataclass(slots=True)
class LocalAgentProvider:
    provider_id: str
    display_name: str
    kind: str
    installed: bool
    executable: str = ""
    version: str = ""
    app_path: str = ""
    llm_capable: bool = False
    default_model: str = ""
    invocation: str = ""
    enabled: bool = False
    status: str = "missing"
    capabilities: list[str] = field(default_factory=list)
    config_refs: list[dict[str, Any]] = field(default_factory=list)
    warnings: list[str] = field(default_factory=list)

    def to_dict(self) -> dict[str, Any]:
        return asdict(self)


def discover_local_agents(workspace_root: str = "", path_env: str | None = None) -> list[dict[str, Any]]:
    home = Path.home()
    workspace = Path(workspace_root).expanduser() if workspace_root else Path.cwd()
    providers = [
        _claude_code_provider(home, workspace, path_env),
        _codex_provider(home, workspace, path_env),
        _cursor_provider(home, workspace, path_env),
        _cursor_agent_provider(home, workspace, path_env),
    ]
    return [item.to_dict() for item in providers]


def runtime_id() -> str:
    raw = f"local-{socket.gethostname() or platform.node() or 'machine'}"
    return re.sub(r"[^a-zA-Z0-9_.-]+", "-", raw).strip("-")[:128] or "local-runtime"


def runtime_name() -> str:
    node = platform.node() or socket.gethostname() or "local"
    return f"Local Agent Runtime ({node})"


def complete_json_with_local_agent(
    *,
    prompt: str,
    payload: dict[str, Any],
    provider_id: str,
    model: str = "",
    timeout_seconds: int = 30,
    workspace_root: str = "",
) -> tuple[dict[str, Any], str, str]:
    provider = _normalize_provider_id(provider_id)
    command_override = os.getenv("LOCAL_AGENT_COMMAND", "").strip()
    if command_override:
        stdout = _run_command(
            [command_override],
            _json_envelope(prompt, payload),
            timeout_seconds,
            workspace_root,
        )
        return _loads_json_object(stdout), "local_agent", provider

    if provider == "auto":
        discovered = discover_local_agents(workspace_root)
        candidates = [item["provider_id"] for item in discovered if item.get("llm_capable")]
        provider = candidates[0] if candidates else ""
    if provider == "claude":
        provider = "claude_code"
    if provider == "codex_cli":
        provider = "codex"
    if provider == "claude_code":
        result = _complete_with_claude(prompt, payload, model, timeout_seconds, workspace_root)
        return result, "local_agent", "claude_code"
    if provider == "codex":
        result = _complete_with_codex(prompt, payload, model, timeout_seconds, workspace_root)
        return result, "local_agent", "codex"
    if provider == "cursor_agent":
        result = _complete_with_cursor_agent(prompt, payload, model, timeout_seconds, workspace_root)
        return result, "local_agent", "cursor_agent"
    raise RuntimeError(f"local agent provider {provider_id!r} is not llm-capable or not installed")


def probe_local_agent(provider_id: str, execute: bool = False, workspace_root: str = "", timeout_seconds: int = 15) -> dict[str, Any]:
    providers = {item["provider_id"]: item for item in discover_local_agents(workspace_root)}
    normalized = _normalize_provider_id(provider_id)
    if normalized == "claude":
        normalized = "claude_code"
    provider = providers.get(normalized)
    if provider is None:
        raise KeyError("local agent provider not found")
    result = dict(provider)
    result["probe_status"] = "installed" if provider.get("installed") else "missing"
    if not execute:
        return result
    if not provider.get("llm_capable") and not os.getenv("LOCAL_AGENT_COMMAND", "").strip():
        result["probe_status"] = "not_llm_capable"
        return result
    try:
        payload, used_provider, used_model = complete_json_with_local_agent(
            prompt="Return JSON only with fields ok and provider.",
            payload={"probe": True},
            provider_id=normalized,
            model=str(provider.get("default_model") or ""),
            timeout_seconds=timeout_seconds,
            workspace_root=workspace_root,
        )
        result["probe_status"] = "ok"
        result["probe_result"] = {"provider": used_provider, "model": used_model, "payload": payload}
    except Exception as exc:
        result["probe_status"] = "failed"
        result["probe_error"] = _redact(str(exc))[:500]
    return result


def _claude_code_provider(home: Path, workspace: Path, path_env: str | None) -> LocalAgentProvider:
    executable = shutil.which("claude", path=path_env) or ""
    installed = bool(executable)
    help_text = _command_output([executable, "--help"], 3) if installed else ""
    version = _first_line(_command_output([executable, "--version"], 3)) if installed else ""
    llm_capable = installed and "--print" in help_text and "--output-format" in help_text
    capabilities = ["mcp_config", "subagents", "code_read"]
    if llm_capable:
        capabilities.append("non_interactive_json")
    return LocalAgentProvider(
        provider_id="claude_code",
        display_name="Claude Code",
        kind="coding_agent",
        installed=installed,
        executable=executable,
        version=version,
        app_path=_existing_path("/Applications/Claude.app"),
        llm_capable=llm_capable,
        default_model=os.getenv("LOCAL_AGENT_CLAUDE_MODEL", "sonnet"),
        invocation="claude --print --bare --tools ''",
        status="available" if installed else "missing",
        capabilities=capabilities,
        config_refs=_config_refs(
            home,
            workspace,
            [
                home / ".claude" / "settings.json",
                home / ".claude.json",
                workspace / ".claude" / "settings.json",
                workspace / ".claude" / "settings.local.json",
                workspace / ".mcp.json",
            ],
        ),
        warnings=[] if llm_capable else ["claude command not found or non-interactive print mode unavailable"],
    )


def _codex_provider(home: Path, workspace: Path, path_env: str | None) -> LocalAgentProvider:
    executable = shutil.which("codex", path=path_env) or ""
    installed = bool(executable)
    help_text = _command_output([executable, "exec", "--help"], 3) if installed else ""
    version = _first_line(_command_output([executable, "--version"], 3)) if installed else ""
    help_lower = help_text.lower()
    llm_capable = installed and ("--output-last-message" in help_text or "--output-schema" in help_text or "non-interactively" in help_lower)
    capabilities = ["mcp_config", "code_read"]
    if llm_capable:
        capabilities.append("non_interactive_json")
    return LocalAgentProvider(
        provider_id="codex",
        display_name="Codex CLI",
        kind="coding_agent",
        installed=installed,
        executable=executable,
        version=version,
        llm_capable=llm_capable,
        default_model=os.getenv("LOCAL_AGENT_CODEX_MODEL", ""),
        invocation="codex exec --sandbox read-only --ephemeral",
        status="available" if installed else "missing",
        capabilities=capabilities,
        config_refs=_config_refs(home, workspace, [home / ".codex" / "config.toml", workspace / ".codex" / "config.toml"]),
        warnings=["codex may read workspace in read-only mode; use only on allowlisted workspaces"] if llm_capable else ["codex exec unavailable"],
    )


def _cursor_provider(home: Path, workspace: Path, path_env: str | None) -> LocalAgentProvider:
    executable = shutil.which("cursor", path=path_env) or ""
    installed = bool(executable) or Path("/Applications/Cursor.app").exists()
    version = _first_line(_command_output([executable, "--version"], 3)) if executable else ""
    return LocalAgentProvider(
        provider_id="cursor",
        display_name="Cursor",
        kind="editor",
        installed=installed,
        executable=executable,
        version=version,
        app_path=_existing_path("/Applications/Cursor.app"),
        llm_capable=False,
        status="editor_only" if installed else "missing",
        capabilities=["mcp_config", "editor"],
        config_refs=_config_refs(home, workspace, [home / ".cursor" / "mcp.json", workspace / ".cursor" / "mcp.json"]),
        warnings=["cursor editor CLI is discoverable but not a stable non-interactive LLM provider; install cursor-agent when available"],
    )


def _cursor_agent_provider(home: Path, workspace: Path, path_env: str | None) -> LocalAgentProvider:
    executable = shutil.which("cursor-agent", path=path_env) or ""
    installed = bool(executable)
    version = _first_line(_command_output([executable, "--version"], 3)) if installed else ""
    help_text = _command_output([executable, "--help"], 3) if installed else ""
    llm_capable = installed and ("--prompt" in help_text or "prompt" in help_text.lower())
    return LocalAgentProvider(
        provider_id="cursor_agent",
        display_name="Cursor Agent",
        kind="coding_agent",
        installed=installed,
        executable=executable,
        version=version,
        llm_capable=llm_capable,
        default_model=os.getenv("LOCAL_AGENT_CURSOR_MODEL", ""),
        invocation="cursor-agent --prompt",
        status="available" if installed else "missing",
        capabilities=["mcp_config", "code_read"] + (["non_interactive_json"] if llm_capable else []),
        config_refs=_config_refs(home, workspace, [home / ".cursor" / "mcp.json", workspace / ".cursor" / "mcp.json"]),
        warnings=[] if llm_capable else ["cursor-agent not found or non-interactive prompt mode unavailable"],
    )


def _complete_with_claude(prompt: str, payload: dict[str, Any], model: str, timeout_seconds: int, workspace_root: str) -> dict[str, Any]:
    executable = shutil.which("claude")
    if not executable:
        raise RuntimeError("claude command not found")
    user_prompt = _json_prompt(prompt, payload)
    args = [
        executable,
        "--print",
        "--bare",
        "--no-session-persistence",
        "--output-format",
        "text",
        "--tools",
        "",
        "--system-prompt",
        prompt,
    ]
    selected_model = _model_or_empty(model, {"auto", "claude_code", "claude"})
    if selected_model:
        args.extend(["--model", selected_model])
    args.append(user_prompt)
    return _loads_json_object(_run_command(args, "", timeout_seconds, workspace_root))


def _complete_with_codex(prompt: str, payload: dict[str, Any], model: str, timeout_seconds: int, workspace_root: str) -> dict[str, Any]:
    executable = shutil.which("codex")
    if not executable:
        raise RuntimeError("codex command not found")
    with tempfile.TemporaryDirectory(prefix="ai-troubleshooter-codex-") as tmp:
        output_path = Path(tmp) / "last_message.json"
        args = [
            executable,
            "exec",
            "--skip-git-repo-check",
            "--ephemeral",
            "--sandbox",
            "read-only",
            "--output-last-message",
            str(output_path),
        ]
        help_text = _command_output([executable, "exec", "--help"], 3)
        if "--ask-for-approval" in help_text:
            args.extend(["--ask-for-approval", "never"])
        selected_model = _model_or_empty(model, {"auto", "codex", "codex_cli"})
        if selected_model:
            args.extend(["--model", selected_model])
        if workspace_root:
            args.extend(["--cd", workspace_root])
        args.append(_json_prompt(prompt, payload))
        stdout = _run_command(args, "", timeout_seconds, workspace_root)
        text = output_path.read_text(encoding="utf-8") if output_path.exists() else stdout
    return _loads_json_object(text)


def _complete_with_cursor_agent(prompt: str, payload: dict[str, Any], model: str, timeout_seconds: int, workspace_root: str) -> dict[str, Any]:
    executable = shutil.which("cursor-agent")
    if not executable:
        raise RuntimeError("cursor-agent command not found")
    args = [
        executable,
        "chat",
        "-p",
        _json_prompt(prompt, payload),
        "--output-format",
        "text",
    ]
    selected_model = _model_or_empty(model, {"auto", "cursor_agent", "cursor"})
    if selected_model:
        args.extend(["--model", selected_model])
    if workspace_root:
        args.extend(["--workspace", workspace_root])
    return _loads_json_object(_run_command(args, "", timeout_seconds, workspace_root))


def _json_prompt(prompt: str, payload: dict[str, Any]) -> str:
    return (
        "You are a production troubleshooting decision LLM. "
        "Return exactly one JSON object and no markdown.\n\n"
        f"System instruction:\n{prompt}\n\n"
        "Input JSON:\n"
        f"{json.dumps(payload, ensure_ascii=False, default=str)}"
    )


def _json_envelope(prompt: str, payload: dict[str, Any]) -> str:
    return json.dumps({"prompt": prompt, "payload": payload}, ensure_ascii=False, default=str)


def _run_command(args: list[str], stdin: str, timeout_seconds: int, workspace_root: str) -> str:
    cwd = workspace_root if workspace_root and Path(workspace_root).exists() else None
    completed = subprocess.run(
        args,
        input=stdin,
        text=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        cwd=cwd,
        timeout=max(1, timeout_seconds),
        check=False,
    )
    if completed.returncode != 0:
        raise RuntimeError(f"local agent command failed exit={completed.returncode}: {_redact(completed.stderr or completed.stdout)}")
    return completed.stdout.strip()


def _loads_json_object(text: str) -> dict[str, Any]:
    cleaned = re.sub(r"^```(?:json)?\s*|\s*```$", "", str(text).strip(), flags=re.IGNORECASE | re.MULTILINE)
    start = cleaned.find("{")
    end = cleaned.rfind("}")
    if start >= 0 and end >= start:
        cleaned = cleaned[start : end + 1]
    value = json.loads(cleaned or "{}")
    if not isinstance(value, dict):
        raise ValueError("local agent returned non-object JSON")
    return value


def _command_output(args: list[str], timeout_seconds: int) -> str:
    if not args or not args[0]:
        return ""
    try:
        completed = subprocess.run(args, text=True, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, timeout=timeout_seconds, check=False)
    except Exception:
        return ""
    return completed.stdout.strip()


def _config_refs(home: Path, workspace: Path, paths: list[Path]) -> list[dict[str, Any]]:
    refs: list[dict[str, Any]] = []
    for path in paths:
        try:
            expanded = path.expanduser()
            if not expanded.exists():
                continue
            refs.append(
                {
                    "path": _display_path(expanded, home, workspace),
                    "scope": "workspace" if _is_relative_to(expanded, workspace) else "user",
                    "exists": True,
                }
            )
        except OSError:
            continue
    return refs


def _display_path(path: Path, home: Path, workspace: Path) -> str:
    if _is_relative_to(path, workspace):
        return str(path)
    if _is_relative_to(path, home):
        return "~/" + str(path.relative_to(home))
    return str(path)


def _is_relative_to(path: Path, parent: Path) -> bool:
    try:
        path.resolve().relative_to(parent.resolve())
        return True
    except ValueError:
        return False


def _existing_path(value: str) -> str:
    return value if Path(value).exists() else ""


def _first_line(text: str) -> str:
    return (text.splitlines()[0].strip() if text.strip() else "")[:200]


def _normalize_provider_id(value: str) -> str:
    return str(value or "auto").strip().lower().replace("-", "_")


def _model_or_empty(value: str, ignored: set[str]) -> str:
    model = str(value or "").strip()
    return "" if not model or model.lower() in ignored else model


def _redact(value: str) -> str:
    text = str(value)
    text = re.sub(r"(?i)(api[_-]?key|token|authorization|password|secret)[^,}\\n]*", r"\1=<redacted>", text)
    text = re.sub(r"Bearer\s+[A-Za-z0-9._\\-]+", "Bearer <redacted>", text)
    return text
