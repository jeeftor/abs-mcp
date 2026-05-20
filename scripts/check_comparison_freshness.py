#!/usr/bin/env python3
"""Warn when the README MCP comparison has not been refreshed recently."""

from __future__ import annotations

import argparse
import datetime as dt
import re
import sys
from pathlib import Path

HEADING_RE = re.compile(
    r"^## AI Generated Comparison - Last updated (?P<date>\d{4}-\d{2}-\d{2})$",
    re.MULTILINE,
)


def main() -> int:
    """Run the comparison freshness check."""
    parser = argparse.ArgumentParser()
    parser.add_argument("--readme", type=Path, default=Path("README.md"))
    parser.add_argument("--max-age-days", type=int, default=7)
    args = parser.parse_args()

    warning = check_freshness(args.readme, args.max_age_days, dt.date.today())
    if warning:
        print(warning, file=sys.stderr)
    return 0


def check_freshness(readme: Path, max_age_days: int, today: dt.date) -> str:
    """Return a non-blocking warning when the comparison is stale or missing."""
    if not readme.exists():
        return f"{readme}: missing README; cannot check MCP comparison freshness"

    match = HEADING_RE.search(readme.read_text(encoding="utf-8"))
    if not match:
        return (
            f"{readme}: no AI Generated Comparison timestamp found. "
            "Run the $abs-mcp-comparison skill to refresh competitor evidence."
        )

    try:
        last_updated = dt.date.fromisoformat(match.group("date"))
    except ValueError:
        return (
            f"{readme}: invalid AI Generated Comparison timestamp "
            f"{match.group('date')!r}. Run $abs-mcp-comparison."
        )

    age_days = (today - last_updated).days
    if age_days > max_age_days:
        return (
            f"{readme}: AI Generated Comparison is {age_days} days old. "
            "Consider running $abs-mcp-comparison to refresh external MCP "
            "server evidence and feature-gap suggestions."
        )
    return ""


if __name__ == "__main__":
    raise SystemExit(main())
