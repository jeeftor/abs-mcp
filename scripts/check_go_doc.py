#!/usr/bin/env python3
"""Check exported Go declarations in non-test files for Godoc comments."""

from __future__ import annotations

import argparse
import re
import sys
from pathlib import Path

DECL_RE = re.compile(r"^(func|type|var|const)\s+(?:\([^)]*\)\s*)?([A-Z][A-Za-z0-9_]*)\b")
SPEC_RE = re.compile(r"^\s*([A-Z][A-Za-z0-9_]*)\b")


def main() -> int:
    """Run the Go doc comment check."""
    parser = argparse.ArgumentParser()
    parser.add_argument("paths", nargs="*", type=Path)
    args = parser.parse_args()

    paths = [
        path
        for path in args.paths
        if path.suffix == ".go" and not path.name.endswith("_test.go") and path.exists()
    ]
    failures: list[str] = []
    for path in paths:
        failures.extend(check_file(path))

    if failures:
        for failure in failures:
            print(failure, file=sys.stderr)
        return 1
    return 0


def check_file(path: Path) -> list[str]:
    """Return doc-comment failures for one Go source file."""
    lines = path.read_text(encoding="utf-8").splitlines()
    failures: list[str] = []
    in_block = False
    block_kind = ""
    block_doc = ""

    for index, line in enumerate(lines):
        stripped = line.strip()
        if not stripped:
            continue

        if in_block:
            if stripped == ")":
                in_block = False
                block_kind = ""
                block_doc = ""
                continue
            spec_match = SPEC_RE.match(line)
            if spec_match:
                name = spec_match.group(1)
                if block_kind in {"const", "var"} and not has_godoc(lines, index, name, block_doc):
                    failures.append(format_failure(path, index, name))
            continue

        decl_match = DECL_RE.match(stripped)
        if decl_match:
            kind, name = decl_match.groups()
            if not has_godoc(lines, index, name, ""):
                failures.append(format_failure(path, index, name))
            if stripped.endswith("(") and kind in {"const", "var"}:
                in_block = True
                block_kind = kind
                block_doc = previous_comment(lines, index)
            continue

        if stripped in {"const (", "var ("}:
            in_block = True
            block_kind = stripped.split()[0]
            block_doc = previous_comment(lines, index)

    return failures


def has_godoc(lines: list[str], index: int, name: str, block_doc: str) -> bool:
    """Return whether a declaration has a comment beginning with its name."""
    doc = previous_comment(lines, index)
    return doc.startswith(f"{name} ") or doc.startswith(f"{name}.") or block_doc.startswith(f"{name} ")


def previous_comment(lines: list[str], index: int) -> str:
    """Return the contiguous line-comment text immediately above index."""
    parts: list[str] = []
    cursor = index - 1
    while cursor >= 0:
        stripped = lines[cursor].strip()
        if not stripped.startswith("//"):
            break
        parts.append(stripped.removeprefix("//").strip())
        cursor -= 1
    parts.reverse()
    return " ".join(parts)


def format_failure(path: Path, index: int, name: str) -> str:
    """Format one doc-comment failure."""
    return f"{path}:{index + 1}: exported declaration {name} needs a Godoc comment"


if __name__ == "__main__":
    raise SystemExit(main())
