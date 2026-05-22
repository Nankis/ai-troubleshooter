from __future__ import annotations

import fnmatch
import json
import os
import re
from dataclasses import dataclass, field
from pathlib import Path
from typing import Any


DEFAULT_ALLOWED_GLOBS = (
    "src/main/java/**",
    "src/main/resources/**",
    "**/*.go",
    "**/*.py",
    "**/*.js",
    "**/*.ts",
    "**/*.yaml",
    "**/*.yml",
    "**/*.properties",
    "**/*.md",
)

DEFAULT_DENY_GLOBS = (
    ".git/**",
    ".env",
    ".env.*",
    "**/.env",
    "**/.env.*",
    "*secret*",
    "**/*secret*",
    "*token*",
    "**/*token*",
    "*credential*",
    "**/*credential*",
    "application-prod.yml",
    "application-prod.yaml",
    "**/application-prod.yml",
    "**/application-prod.yaml",
    "application-production.yml",
    "application-production.yaml",
    "**/application-production.yml",
    "**/application-production.yaml",
    "**/*.pem",
    "**/*.key",
    "**/*.p12",
    "**/*.jks",
)

TEXT_FILE_MAX_BYTES = 256 * 1024


@dataclass(frozen=True, slots=True)
class LocalRepoConfig:
    service_name: str
    repo_path: Path
    allowed_globs: tuple[str, ...] = DEFAULT_ALLOWED_GLOBS
    deny_globs: tuple[str, ...] = DEFAULT_DENY_GLOBS

    @classmethod
    def from_dict(cls, service_name: str, value: dict[str, Any]) -> "LocalRepoConfig":
        allowed = tuple(str(item) for item in value.get("allowed_globs") or DEFAULT_ALLOWED_GLOBS)
        deny = tuple(str(item) for item in value.get("deny_globs") or DEFAULT_DENY_GLOBS)
        return cls(
            service_name=service_name,
            repo_path=Path(str(value.get("repo_path", ""))).expanduser(),
            allowed_globs=allowed,
            deny_globs=deny + tuple(item for item in DEFAULT_DENY_GLOBS if item not in deny),
        )


@dataclass(slots=True)
class LocalCodeHit:
    file_path: str
    matched_terms: list[str] = field(default_factory=list)
    line_numbers: list[int] = field(default_factory=list)

    def to_dict(self) -> dict[str, Any]:
        return {
            "file_path": self.file_path,
            "matched_terms": self.matched_terms,
            "line_numbers": self.line_numbers,
        }


@dataclass(slots=True)
class LocalCodeInspection:
    service_name: str
    repo_id: str
    status: str
    summary: str
    hits: list[LocalCodeHit] = field(default_factory=list)
    skipped_denied_files: int = 0
    scanned_files: int = 0
    risks: list[str] = field(default_factory=list)

    def evidence(self) -> list[dict[str, Any]]:
        return [hit.to_dict() for hit in self.hits]


class LocalCodeInspector:
    def __init__(self, repos: dict[str, LocalRepoConfig] | None = None, enabled: bool = True) -> None:
        self.repos = repos or {}
        self.enabled = enabled

    @classmethod
    def from_env(cls) -> "LocalCodeInspector":
        raw = os.getenv("LOCAL_CODE_REPOS_JSON", "").strip()
        if not raw:
            return cls(enabled=False)
        data = json.loads(raw)
        if not isinstance(data, dict):
            raise ValueError("LOCAL_CODE_REPOS_JSON must be a JSON object")
        repos: dict[str, LocalRepoConfig] = {}
        for service_name, value in data.items():
            if not isinstance(value, dict):
                raise ValueError(f"repo config for {service_name} must be an object")
            config = LocalRepoConfig.from_dict(str(service_name), value)
            repos[config.service_name] = config
        return cls(repos=repos, enabled=True)

    def inspect(
        self,
        service_name: str,
        query_text: str,
        repo_hint: str = "",
        max_hits: int = 8,
    ) -> LocalCodeInspection:
        repo = self._resolve_repo(service_name, repo_hint)
        if not self.enabled:
            return LocalCodeInspection(
                service_name=service_name,
                repo_id=service_name or repo_hint,
                status="disabled",
                summary="本地代码检查未启用，需要配置 LOCAL_CODE_REPOS_JSON。",
                risks=["local_code_debug_disabled"],
            )
        if repo is None:
            return LocalCodeInspection(
                service_name=service_name,
                repo_id=service_name or repo_hint,
                status="no_mapping",
                summary="没有找到 service_name/repo_hint 对应的本地仓库映射。",
                risks=["local_repo_mapping_missing"],
            )

        root = repo.repo_path.resolve()
        if not root.exists() or not root.is_dir():
            return LocalCodeInspection(
                service_name=service_name,
                repo_id=repo.service_name,
                status="repo_unavailable",
                summary="本地仓库路径不存在或不是目录。",
                risks=["local_repo_unavailable"],
            )

        terms = self._query_terms(query_text)
        if not terms:
            return LocalCodeInspection(
                service_name=service_name,
                repo_id=repo.service_name,
                status="no_query_terms",
                summary="没有足够关键词执行本地代码搜索。",
                risks=["local_code_query_empty"],
            )

        hits: list[LocalCodeHit] = []
        skipped_denied = 0
        scanned = 0
        for path in self._iter_allowed_files(root, repo):
            if not self._inside_root(root, path):
                skipped_denied += 1
                continue
            relative = self._relative_posix(root, path)
            if self._is_denied(relative, repo.deny_globs):
                skipped_denied += 1
                continue
            if path.stat().st_size > TEXT_FILE_MAX_BYTES:
                continue
            scanned += 1
            hit = self._scan_file(root, path, terms)
            if hit is not None:
                hits.append(hit)
                if len(hits) >= max_hits:
                    break

        if hits:
            return LocalCodeInspection(
                service_name=service_name,
                repo_id=repo.service_name,
                status="matched",
                summary=f"本地代码只读搜索命中 {len(hits)} 个文件；结果仅包含相对路径、命中词和行号。",
                hits=hits,
                skipped_denied_files=skipped_denied,
                scanned_files=scanned,
            )

        return LocalCodeInspection(
            service_name=service_name,
            repo_id=repo.service_name,
            status="no_match",
            summary="本地代码只读搜索未命中相关文件。",
            skipped_denied_files=skipped_denied,
            scanned_files=scanned,
        )

    def _resolve_repo(self, service_name: str, repo_hint: str) -> LocalRepoConfig | None:
        for key in (service_name, repo_hint):
            if key and key in self.repos:
                return self.repos[key]
        return None

    def _query_terms(self, text: str) -> list[str]:
        terms: list[str] = []
        seen: set[str] = set()
        for item in re.findall(r"[A-Za-z0-9_.$:-]{3,}", text.lower()):
            normalized = item.strip("_.$:-")
            if len(normalized) < 3 or normalized in seen:
                continue
            seen.add(normalized)
            terms.append(normalized)
            if len(terms) >= 16:
                break
        return terms

    def _iter_allowed_files(self, root: Path, repo: LocalRepoConfig):
        seen: set[Path] = set()
        for pattern in repo.allowed_globs:
            for path in root.glob(pattern):
                if path in seen or not path.is_file():
                    continue
                seen.add(path)
                yield path

    def _scan_file(self, root: Path, path: Path, terms: list[str]) -> LocalCodeHit | None:
        try:
            content = path.read_text(encoding="utf-8", errors="ignore")
        except OSError:
            return None
        lower_content = content.lower()
        matched_terms = [term for term in terms if term in lower_content]
        if not matched_terms:
            return None

        line_numbers: list[int] = []
        for idx, line in enumerate(content.splitlines(), start=1):
            lower_line = line.lower()
            if any(term in lower_line for term in matched_terms):
                line_numbers.append(idx)
                if len(line_numbers) >= 8:
                    break

        return LocalCodeHit(
            file_path=self._relative_posix(root, path),
            matched_terms=matched_terms[:8],
            line_numbers=line_numbers,
        )

    def _is_denied(self, relative_path: str, deny_globs: tuple[str, ...]) -> bool:
        lowered = relative_path.lower()
        return any(fnmatch.fnmatch(lowered, pattern.lower()) for pattern in deny_globs)

    def _inside_root(self, root: Path, path: Path) -> bool:
        try:
            path.resolve().relative_to(root)
        except ValueError:
            return False
        return True

    def _relative_posix(self, root: Path, path: Path) -> str:
        return path.resolve().relative_to(root).as_posix()
