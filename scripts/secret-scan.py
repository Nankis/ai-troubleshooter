#!/usr/bin/env python3
from __future__ import annotations

import argparse
import pathlib
import re
import subprocess
import sys


ALLOW_LINE = re.compile(
    r"(?i)(replace-with|placeholder|your-|dummy|example|mock|fixture|test-key|test-api-key|unit-test|sk-\.\.\.|xxx|changeme|\$\{[A-Z0-9_]+:?[^}]*\}|\$[A-Z0-9_]+)"
)

PATTERNS: list[tuple[str, re.Pattern[str]]] = [
    ("private key", re.compile(r"-----BEGIN [A-Z ]*PRIVATE KEY-----")),
    ("openai/dashscope style key", re.compile(r"\bsk-[A-Za-z0-9][A-Za-z0-9_-]{20,}\b")),
    ("aliyun access key id", re.compile(r"\bLTAI[A-Za-z0-9]{16,}\b")),
    ("aws access key id", re.compile(r"\bAKIA[0-9A-Z]{16}\b")),
    ("quoted secret assignment", re.compile(r"(?i)\b(token|api[_-]?key|secret|password|passwd)\b\s*[:=]\s*['\"][^'\"\s]{8,}")),
    ("dsn password", re.compile(r"(?i)([a-z0-9_]+):([^@\s:]{8,})@tcp\(|://[^/\s:@]+:[^/\s:@]{8,}@")),
]

SKIP_SUFFIXES = {
    ".png",
    ".jpg",
    ".jpeg",
    ".gif",
    ".webp",
    ".ico",
    ".pdf",
    ".zip",
    ".gz",
    ".sum",
}

SKIP_PARTS = {
    ".git",
    "node_modules",
    "target",
    "dist",
    "build",
    "vendor",
}


def git_lines(args: list[str]) -> list[str]:
    result = subprocess.run(["git", *args], text=True, capture_output=True, check=True)
    return [line.strip() for line in result.stdout.splitlines() if line.strip()]


def files_to_scan(mode: str) -> list[pathlib.Path]:
    if mode == "staged":
        names = git_lines(["diff", "--cached", "--name-only", "--diff-filter=ACMR"])
    else:
        names = git_lines(["ls-files"])
    return [pathlib.Path(name) for name in names if should_scan(pathlib.Path(name))]


def should_scan(path: pathlib.Path) -> bool:
    if any(part in SKIP_PARTS for part in path.parts):
        return False
    if path.suffix.lower() in SKIP_SUFFIXES:
        return False
    return path.exists() and path.is_file()


def scan_file(path: pathlib.Path) -> list[str]:
    findings: list[str] = []
    try:
        text = path.read_text(encoding="utf-8")
    except UnicodeDecodeError:
        return findings
    for lineno, line in enumerate(text.splitlines(), start=1):
        if ALLOW_LINE.search(line):
            continue
        for label, pattern in PATTERNS:
            if pattern.search(line):
                findings.append(f"{path}:{lineno}: possible {label}")
                break
    return findings


def main() -> int:
    parser = argparse.ArgumentParser(description="Scan repository files for likely secrets before commit/push.")
    parser.add_argument("--mode", choices=("staged", "all"), default="staged")
    args = parser.parse_args()

    findings: list[str] = []
    for path in files_to_scan(args.mode):
        findings.extend(scan_file(path))

    if findings:
        print("Secret scan failed. Remove secrets or replace with env placeholders:", file=sys.stderr)
        for finding in findings:
            print(f"  {finding}", file=sys.stderr)
        return 1
    print(f"Secret scan passed ({args.mode}).")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
