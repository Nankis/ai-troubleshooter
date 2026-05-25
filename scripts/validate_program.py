#!/usr/bin/env python3
"""Validate Program evidence structure for ai-troubleshooter."""

from __future__ import annotations

import argparse
import re
import sys
from pathlib import Path


REQUIRED_FILES = ("PROGRAM.md", "STATUS.yml", "TASKS.md", "EVIDENCE.md", "HANDOFF.md")
COMPLETED_EXTRA_FILES = ("RESULT.md",)
EVIDENCE_SECTIONS = ("Evidence 索引", "命令验证", "现场验证", "覆盖映射", "未验证项", "已知噪音")


def main() -> int:
    parser = argparse.ArgumentParser(description="Validate Program files.")
    parser.add_argument("programs", nargs="+", help="Program directories to validate")
    parser.add_argument("--strict-completed", action="store_true", help="Require RESULT.md only when status is completed")
    args = parser.parse_args()
    errors: list[str] = []
    for raw in args.programs:
        errors.extend(validate_program(Path(raw), strict_completed=args.strict_completed))
    if errors:
        for error in errors:
            print(error, file=sys.stderr)
        return 1
    print(f"validated {len(args.programs)} program(s)")
    return 0


def validate_program(path: Path, *, strict_completed: bool) -> list[str]:
    errors: list[str] = []
    if not path.is_dir():
        return [f"{path}: not a directory"]
    for name in REQUIRED_FILES:
        if not (path / name).is_file():
            errors.append(f"{path}: missing {name}")
    status_text = read_text(path / "STATUS.yml")
    status = parse_status(status_text)
    if status == "completed":
        for name in COMPLETED_EXTRA_FILES:
            if not (path / name).is_file():
                errors.append(f"{path}: completed Program missing {name}")
    elif not strict_completed and not (path / "RESULT.md").is_file():
        pass
    handoff = read_text(path / "HANDOFF.md").strip()
    if len(handoff) < 80:
        errors.append(f"{path}: HANDOFF.md is too small to resume from")
    evidence = read_text(path / "EVIDENCE.md")
    for section in EVIDENCE_SECTIONS:
        if f"## {section}" not in evidence:
            errors.append(f"{path}: EVIDENCE.md missing section {section}")
    return errors


def parse_status(text: str) -> str:
    match = re.search(r"^status:\s*([A-Za-z0-9_-]+)\s*$", text, flags=re.MULTILINE)
    return match.group(1).strip() if match else ""


def read_text(path: Path) -> str:
    if not path.is_file():
        return ""
    return path.read_text(encoding="utf-8")


if __name__ == "__main__":
    raise SystemExit(main())
