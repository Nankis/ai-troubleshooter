from __future__ import annotations

import ast
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
MAX_SYMBOLS_PER_HIT = 8
MAX_CALL_EDGES_PER_HIT = 12
CALL_IGNORED_NAMES = {
    "if",
    "for",
    "while",
    "switch",
    "catch",
    "return",
    "throw",
    "new",
    "else",
    "do",
    "try",
    "finally",
    "synchronized",
    "assert",
    "sizeof",
    "typeof",
    "await",
}


@dataclass(slots=True)
class CodeSymbol:
    name: str
    kind: str
    file_path: str
    line_number: int
    language: str
    owner: str = ""

    def search_text(self) -> str:
        return " ".join((self.name, self.kind, self.owner, self.language)).lower()

    def simple_name(self) -> str:
        return self.name.rsplit(".", 1)[-1]

    def to_dict(self) -> dict[str, Any]:
        value: dict[str, Any] = {
            "name": self.name,
            "kind": self.kind,
            "file_path": self.file_path,
            "line_number": self.line_number,
            "language": self.language,
        }
        if self.owner:
            value["owner"] = self.owner
        return value


@dataclass(slots=True)
class CallEdge:
    caller: str
    callee: str
    file_path: str
    line_number: int
    language: str

    def search_text(self) -> str:
        return " ".join((self.caller, self.callee, self.language)).lower()

    def to_dict(self) -> dict[str, Any]:
        return {
            "caller": self.caller,
            "callee": self.callee,
            "file_path": self.file_path,
            "line_number": self.line_number,
            "language": self.language,
        }


@dataclass(slots=True)
class ScannedFile:
    path: Path
    relative_path: str
    content: str
    language: str
    symbols: list[CodeSymbol] = field(default_factory=list)
    call_edges: list[CallEdge] = field(default_factory=list)


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
    symbols: list[CodeSymbol] = field(default_factory=list)
    call_edges: list[CallEdge] = field(default_factory=list)
    analysis_modes: list[str] = field(default_factory=list)

    def to_dict(self) -> dict[str, Any]:
        value: dict[str, Any] = {
            "file_path": self.file_path,
            "matched_terms": self.matched_terms,
            "line_numbers": self.line_numbers,
        }
        if self.symbols:
            value["symbols"] = [symbol.to_dict() for symbol in self.symbols]
        if self.call_edges:
            value["call_edges"] = [edge.to_dict() for edge in self.call_edges]
        if self.analysis_modes:
            value["analysis_modes"] = self.analysis_modes
        return value


@dataclass(slots=True)
class LocalCodeInspection:
    service_name: str
    repo_id: str
    status: str
    summary: str
    hits: list[LocalCodeHit] = field(default_factory=list)
    skipped_denied_files: int = 0
    scanned_files: int = 0
    symbol_count: int = 0
    call_edge_count: int = 0
    analysis_modes: list[str] = field(default_factory=list)
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

        scanned_files: list[ScannedFile] = []
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
            scanned_file = self._read_and_analyze_file(root, path)
            if scanned_file is not None:
                scanned_files.append(scanned_file)

        hits = self._build_hits(scanned_files, terms, max_hits)
        symbol_count = sum(len(item.symbols) for item in scanned_files)
        call_edge_count = sum(len(item.call_edges) for item in scanned_files)
        analysis_modes = self._analysis_modes(symbol_count, call_edge_count)

        if hits:
            return LocalCodeInspection(
                service_name=service_name,
                repo_id=repo.service_name,
                status="matched",
                summary=(
                    f"本地代码只读分析命中 {len(hits)} 个文件；"
                    "结果仅包含相对路径、命中词、符号、调用边和行号。"
                ),
                hits=hits,
                skipped_denied_files=skipped_denied,
                scanned_files=scanned,
                symbol_count=symbol_count,
                call_edge_count=call_edge_count,
                analysis_modes=analysis_modes,
            )

        return LocalCodeInspection(
            service_name=service_name,
            repo_id=repo.service_name,
            status="no_match",
            summary="本地代码只读分析未命中相关文件。",
            skipped_denied_files=skipped_denied,
            scanned_files=scanned,
            symbol_count=symbol_count,
            call_edge_count=call_edge_count,
            analysis_modes=analysis_modes,
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
            compact = self._compact(normalized)
            if len(compact) >= 3 and compact not in seen:
                seen.add(compact)
                terms.append(compact)
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

    def _read_and_analyze_file(self, root: Path, path: Path) -> ScannedFile | None:
        try:
            content = path.read_text(encoding="utf-8", errors="ignore")
        except OSError:
            return None
        relative = self._relative_posix(root, path)
        language = self._language_for_path(path)
        symbols, call_edges = self._analyze_content(relative, language, content)
        return ScannedFile(
            path=path,
            relative_path=relative,
            content=content,
            language=language,
            symbols=symbols,
            call_edges=call_edges,
        )

    def _build_hits(self, scanned_files: list[ScannedFile], terms: list[str], max_hits: int) -> list[LocalCodeHit]:
        hits_by_file: dict[str, LocalCodeHit] = {}
        matched_symbol_names: set[str] = set()

        for scanned_file in scanned_files:
            hit = self._match_scanned_file(scanned_file, terms)
            if hit is None:
                continue
            hits_by_file[hit.file_path] = hit
            for symbol in hit.symbols:
                matched_symbol_names.add(self._compact(symbol.simple_name()))
                matched_symbol_names.add(self._compact(symbol.name))
            for edge in hit.call_edges:
                matched_symbol_names.add(self._compact(edge.caller.rsplit(".", 1)[-1]))
                matched_symbol_names.add(self._compact(edge.callee.rsplit(".", 1)[-1]))

        self._add_call_graph_context(scanned_files, hits_by_file, matched_symbol_names)
        hits = sorted(hits_by_file.values(), key=self._hit_rank, reverse=True)
        return hits[:max(1, max_hits)]

    def _match_scanned_file(self, scanned_file: ScannedFile, terms: list[str]) -> LocalCodeHit | None:
        matched_terms = [term for term in terms if self._matches_text(scanned_file.content, [term])]
        symbols = [symbol for symbol in scanned_file.symbols if self._matches_text(symbol.search_text(), terms)]
        call_edges = [edge for edge in scanned_file.call_edges if self._matches_text(self._edge_match_text(edge), terms)]
        if not matched_terms and not symbols and not call_edges:
            return None

        line_numbers: list[int] = []
        for idx, line in enumerate(scanned_file.content.splitlines(), start=1):
            if self._matches_text(line, matched_terms):
                line_numbers.append(idx)
                if len(line_numbers) >= 8:
                    break

        return LocalCodeHit(
            file_path=scanned_file.relative_path,
            matched_terms=matched_terms[:8],
            line_numbers=line_numbers,
            symbols=symbols[:MAX_SYMBOLS_PER_HIT],
            call_edges=call_edges[:MAX_CALL_EDGES_PER_HIT],
            analysis_modes=self._hit_modes(bool(matched_terms), bool(symbols), bool(call_edges)),
        )

    def _add_call_graph_context(
        self,
        scanned_files: list[ScannedFile],
        hits_by_file: dict[str, LocalCodeHit],
        matched_symbol_names: set[str],
    ) -> None:
        if not matched_symbol_names:
            return
        files_by_path = {item.relative_path: item for item in scanned_files}
        for scanned_file in scanned_files:
            for edge in scanned_file.call_edges:
                caller = self._compact(edge.caller.rsplit(".", 1)[-1])
                callee = self._compact(edge.callee.rsplit(".", 1)[-1])
                if caller not in matched_symbol_names and callee not in matched_symbol_names:
                    continue
                hit = hits_by_file.get(edge.file_path)
                if hit is None:
                    source = files_by_path.get(edge.file_path)
                    hit = LocalCodeHit(
                        file_path=edge.file_path,
                        matched_terms=["call_graph_context"],
                        line_numbers=[edge.line_number],
                        symbols=(source.symbols[:4] if source else []),
                        analysis_modes=["call_graph"],
                    )
                    hits_by_file[edge.file_path] = hit
                if len(hit.call_edges) >= MAX_CALL_EDGES_PER_HIT:
                    continue
                if not any(existing.caller == edge.caller and existing.callee == edge.callee and existing.line_number == edge.line_number for existing in hit.call_edges):
                    hit.call_edges.append(edge)
                if "call_graph" not in hit.analysis_modes:
                    hit.analysis_modes.append("call_graph")

    def _hit_rank(self, hit: LocalCodeHit) -> tuple[int, int, int, int, int]:
        has_structure = 1 if hit.symbols or hit.call_edges else 0
        return (has_structure, len(hit.matched_terms), len(hit.call_edges), len(hit.symbols), len(hit.line_numbers))

    def _edge_match_text(self, edge: CallEdge) -> str:
        caller_method = edge.caller.rsplit(".", 1)[-1]
        return " ".join((caller_method, edge.callee, edge.language)).lower()

    def _analysis_modes(self, symbol_count: int, call_edge_count: int) -> list[str]:
        modes = ["keyword"]
        if symbol_count:
            modes.extend(["language_structure_tree", "symbol_index"])
        if call_edge_count:
            modes.append("call_graph")
        return modes

    def _hit_modes(self, has_keyword: bool, has_symbols: bool, has_call_edges: bool) -> list[str]:
        modes: list[str] = []
        if has_keyword:
            modes.append("keyword")
        if has_symbols:
            modes.extend(["language_structure_tree", "symbol_index"])
        if has_call_edges:
            modes.append("call_graph")
        return modes

    def _matches_text(self, text: str, terms: list[str]) -> bool:
        if not terms:
            return False
        lowered = text.lower()
        compacted = self._compact(lowered)
        return any(term in lowered or self._compact(term) in compacted for term in terms)

    def _compact(self, value: str) -> str:
        return re.sub(r"[^a-z0-9]", "", value.lower())

    def _language_for_path(self, path: Path) -> str:
        suffix = path.suffix.lower()
        if suffix == ".java":
            return "java"
        if suffix == ".py":
            return "python"
        if suffix == ".go":
            return "go"
        if suffix in {".ts", ".tsx"}:
            return "typescript"
        if suffix in {".js", ".jsx", ".mjs", ".cjs"}:
            return "javascript"
        return "text"

    def _analyze_content(self, relative_path: str, language: str, content: str) -> tuple[list[CodeSymbol], list[CallEdge]]:
        if language == "python":
            return self._analyze_python(relative_path, content)
        if language == "java":
            return self._analyze_java(relative_path, content)
        if language == "go":
            return self._analyze_go(relative_path, content)
        if language in {"typescript", "javascript"}:
            return self._analyze_js_ts(relative_path, language, content)
        return [], []

    def _analyze_python(self, relative_path: str, content: str) -> tuple[list[CodeSymbol], list[CallEdge]]:
        try:
            tree = ast.parse(content)
        except SyntaxError:
            return [], []

        symbols: list[CodeSymbol] = []
        call_edges: list[CallEdge] = []

        class Visitor(ast.NodeVisitor):
            def __init__(self) -> None:
                self.owner_stack: list[str] = []
                self.current_callable: str = ""

            def visit_ClassDef(self, node: ast.ClassDef) -> None:  # noqa: N802
                owner = ".".join(self.owner_stack)
                name = ".".join([*self.owner_stack, node.name]) if self.owner_stack else node.name
                symbols.append(CodeSymbol(name=name, kind="class", file_path=relative_path, line_number=node.lineno, language="python", owner=owner))
                self.owner_stack.append(node.name)
                self.generic_visit(node)
                self.owner_stack.pop()

            def visit_FunctionDef(self, node: ast.FunctionDef) -> None:  # noqa: N802
                self._visit_function(node, "function")

            def visit_AsyncFunctionDef(self, node: ast.AsyncFunctionDef) -> None:  # noqa: N802
                self._visit_function(node, "function")

            def visit_Call(self, node: ast.Call) -> None:  # noqa: N802
                if self.current_callable:
                    callee = python_callee_name(node.func)
                    if callee:
                        call_edges.append(
                            CallEdge(
                                caller=self.current_callable,
                                callee=callee,
                                file_path=relative_path,
                                line_number=getattr(node, "lineno", 0) or 0,
                                language="python",
                            )
                        )
                self.generic_visit(node)

            def _visit_function(self, node: ast.FunctionDef | ast.AsyncFunctionDef, kind: str) -> None:
                owner = ".".join(self.owner_stack)
                name = ".".join([*self.owner_stack, node.name]) if self.owner_stack else node.name
                symbols.append(CodeSymbol(name=name, kind=kind, file_path=relative_path, line_number=node.lineno, language="python", owner=owner))
                previous = self.current_callable
                self.current_callable = name
                self.owner_stack.append(node.name)
                self.generic_visit(node)
                self.owner_stack.pop()
                self.current_callable = previous

        def python_callee_name(node: ast.AST) -> str:
            if isinstance(node, ast.Name):
                return node.id
            if isinstance(node, ast.Attribute):
                parent = python_callee_name(node.value)
                return f"{parent}.{node.attr}" if parent else node.attr
            if isinstance(node, ast.Call):
                return python_callee_name(node.func)
            return ""

        Visitor().visit(tree)
        return symbols, call_edges

    def _analyze_java(self, relative_path: str, content: str) -> tuple[list[CodeSymbol], list[CallEdge]]:
        symbols: list[CodeSymbol] = []
        call_edges: list[CallEdge] = []
        current_class = ""
        current_method = ""
        brace_depth = 0
        class_re = re.compile(r"\b(class|interface|enum|record)\s+([A-Za-z_][A-Za-z0-9_]*)")
        method_re = re.compile(
            r"^\s*(?:(?:public|private|protected|static|final|synchronized|abstract|native|default)\s+)*"
            r"[A-Za-z_][A-Za-z0-9_<>\[\],.? extends super]*\s+([A-Za-z_][A-Za-z0-9_]*)\s*\([^;{}]*\)\s*(?:throws [^{]+)?\{?"
        )

        for line_number, line in enumerate(content.splitlines(), start=1):
            class_match = class_re.search(line)
            if class_match:
                current_class = class_match.group(2)
                symbols.append(CodeSymbol(name=current_class, kind=class_match.group(1), file_path=relative_path, line_number=line_number, language="java"))

            if not current_method:
                method_match = method_re.search(line)
                if method_match and method_match.group(1) not in CALL_IGNORED_NAMES:
                    method_name = method_match.group(1)
                    owner = current_class
                    symbol_name = f"{owner}.{method_name}" if owner else method_name
                    symbols.append(CodeSymbol(name=symbol_name, kind="method", file_path=relative_path, line_number=line_number, language="java", owner=owner))
                    current_method = symbol_name
                    brace_depth = 0

            if current_method:
                for callee in self._call_names(line):
                    if callee == current_method.rsplit(".", 1)[-1]:
                        continue
                    call_edges.append(CallEdge(caller=current_method, callee=callee, file_path=relative_path, line_number=line_number, language="java"))
                brace_depth += line.count("{") - line.count("}")
                if brace_depth <= 0 and ("{" in line or "}" in line):
                    current_method = ""

        return symbols, call_edges

    def _analyze_go(self, relative_path: str, content: str) -> tuple[list[CodeSymbol], list[CallEdge]]:
        symbols: list[CodeSymbol] = []
        call_edges: list[CallEdge] = []
        current_func = ""
        brace_depth = 0
        func_re = re.compile(r"\bfunc\s+(?:\([^)]*\)\s*)?([A-Za-z_][A-Za-z0-9_]*)\s*\(")
        type_re = re.compile(r"\btype\s+([A-Za-z_][A-Za-z0-9_]*)\s+(struct|interface)\b")

        for line_number, line in enumerate(content.splitlines(), start=1):
            type_match = type_re.search(line)
            if type_match:
                symbols.append(CodeSymbol(name=type_match.group(1), kind=type_match.group(2), file_path=relative_path, line_number=line_number, language="go"))

            if not current_func:
                func_match = func_re.search(line)
                if func_match:
                    current_func = func_match.group(1)
                    symbols.append(CodeSymbol(name=current_func, kind="function", file_path=relative_path, line_number=line_number, language="go"))
                    brace_depth = 0

            if current_func:
                for callee in self._call_names(line):
                    if callee == current_func:
                        continue
                    call_edges.append(CallEdge(caller=current_func, callee=callee, file_path=relative_path, line_number=line_number, language="go"))
                brace_depth += line.count("{") - line.count("}")
                if brace_depth <= 0 and ("{" in line or "}" in line):
                    current_func = ""

        return symbols, call_edges

    def _analyze_js_ts(self, relative_path: str, language: str, content: str) -> tuple[list[CodeSymbol], list[CallEdge]]:
        symbols: list[CodeSymbol] = []
        call_edges: list[CallEdge] = []
        current_callable = ""
        brace_depth = 0
        class_re = re.compile(r"\bclass\s+([A-Za-z_$][A-Za-z0-9_$]*)")
        function_re = re.compile(r"\b(?:async\s+)?function\s+([A-Za-z_$][A-Za-z0-9_$]*)\s*\(")
        const_function_re = re.compile(r"\b(?:const|let|var)\s+([A-Za-z_$][A-Za-z0-9_$]*)\s*=\s*(?:async\s*)?(?:\([^)]*\)|[A-Za-z_$][A-Za-z0-9_$]*)\s*=>")

        for line_number, line in enumerate(content.splitlines(), start=1):
            class_match = class_re.search(line)
            if class_match:
                symbols.append(CodeSymbol(name=class_match.group(1), kind="class", file_path=relative_path, line_number=line_number, language=language))

            if not current_callable:
                function_match = function_re.search(line) or const_function_re.search(line)
                if function_match:
                    current_callable = function_match.group(1)
                    symbols.append(CodeSymbol(name=current_callable, kind="function", file_path=relative_path, line_number=line_number, language=language))
                    brace_depth = 0

            if current_callable:
                for callee in self._call_names(line):
                    if callee == current_callable:
                        continue
                    call_edges.append(CallEdge(caller=current_callable, callee=callee, file_path=relative_path, line_number=line_number, language=language))
                brace_depth += line.count("{") - line.count("}")
                if brace_depth <= 0 and ("{" in line or "}" in line):
                    current_callable = ""

        return symbols, call_edges

    def _call_names(self, line: str) -> list[str]:
        stripped = re.sub(r"//.*$", "", line)
        stripped = re.sub(r"#.*$", "", stripped)
        names: list[str] = []
        seen: set[str] = set()
        for match in re.finditer(r"\b([A-Za-z_][A-Za-z0-9_]*)\s*\(", stripped):
            name = match.group(1)
            if name in CALL_IGNORED_NAMES or name in seen:
                continue
            seen.add(name)
            names.append(name)
        return names

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
