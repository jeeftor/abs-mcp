#!/usr/bin/env python3
"""Compare two generated Audiobookshelf API inventory files."""

from __future__ import annotations

import argparse
import json
from pathlib import Path
from typing import Any


RouteMap = dict[tuple[str, str], dict[str, Any]]


def parse_args() -> argparse.Namespace:
    """Parse CLI arguments."""

    parser = argparse.ArgumentParser(description="Diff Audiobookshelf API inventories.")
    parser.add_argument(
        "--baseline",
        type=Path,
        default=Path("docs/api-inventory/baseline/abs-api-inventory.json"),
        help="Baseline inventory JSON.",
    )
    parser.add_argument(
        "--generated",
        type=Path,
        default=Path("docs/api-inventory/generated/abs-api-inventory.json"),
        help="Generated inventory JSON.",
    )
    parser.add_argument(
        "--fail-on-change",
        action="store_true",
        help="Exit with status 1 when route changes are detected.",
    )
    return parser.parse_args()


def load_inventory(path: Path) -> dict[str, Any]:
    """Load an inventory JSON file."""

    with path.open(encoding="utf-8") as inventory_file:
        data = json.load(inventory_file)
    if not isinstance(data, dict):
        raise TypeError(f"inventory must be a JSON object: {path}")
    return data


def route_map(inventory: dict[str, Any]) -> RouteMap:
    """Index inventory routes by method and path."""

    routes = inventory.get("routes")
    if not isinstance(routes, list):
        raise TypeError("inventory routes must be a list")

    indexed: RouteMap = {}
    for route in routes:
        if not isinstance(route, dict):
            raise TypeError("route must be an object")
        method = route.get("method")
        path = route.get("path")
        if not isinstance(method, str) or not isinstance(path, str):
            raise TypeError("route method and path must be strings")
        indexed[(method, path)] = route
    return indexed


def changed_fields(before: dict[str, Any], after: dict[str, Any]) -> list[str]:
    """Return route fields that changed in a meaningful way."""

    fields = [
        "group",
        "handler_controller",
        "handler_method",
        "middleware",
        "mutates",
        "admin_likely",
    ]
    return [field for field in fields if before.get(field) != after.get(field)]


def route_label(route_key: tuple[str, str]) -> str:
    """Format a route key for output."""

    method, path = route_key
    return f"{method} {path}"


def print_section(title: str, rows: list[str]) -> None:
    """Print a titled diff section."""

    print(f"\n## {title}")
    if not rows:
        print("none")
        return
    for row in rows:
        print(f"- {row}")


def main() -> None:
    """Run the inventory diff."""

    args = parse_args()
    baseline = route_map(load_inventory(args.baseline))
    generated = route_map(load_inventory(args.generated))

    baseline_keys = set(baseline)
    generated_keys = set(generated)

    added = sorted(generated_keys - baseline_keys)
    removed = sorted(baseline_keys - generated_keys)
    changed: list[str] = []

    for key in sorted(baseline_keys & generated_keys):
        fields = changed_fields(baseline[key], generated[key])
        if fields:
            changed.append(f"{route_label(key)} ({', '.join(fields)})")

    print("# Audiobookshelf API Inventory Diff")
    print_section("Added Routes", [route_label(key) for key in added])
    print_section("Removed Routes", [route_label(key) for key in removed])
    print_section("Changed Routes", changed)

    change_count = len(added) + len(removed) + len(changed)
    print(f"\nTotal changes: {change_count}")
    if args.fail_on_change and change_count:
        raise SystemExit(1)


if __name__ == "__main__":
    main()
