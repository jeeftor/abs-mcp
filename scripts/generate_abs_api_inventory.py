#!/usr/bin/env python3
"""Generate an Audiobookshelf API route inventory from ApiRouter.js."""

from __future__ import annotations

import argparse
import json
import re
from collections import Counter
from dataclasses import asdict, dataclass
from pathlib import Path
from typing import Sequence


ROUTER_RELATIVE_PATH = Path("server/routers/ApiRouter.js")
ROUTE_METHODS = {"get", "post", "patch", "delete", "put"}
MUTATING_METHODS = {"post", "patch", "delete", "put"}


@dataclass(frozen=True)
class RouteBinding:
    """A single Express route binding extracted from Audiobookshelf source."""

    method: str
    path: str
    path_kind: str
    group: str
    handler_controller: str
    handler_method: str
    middleware: list[str]
    mutates: bool
    admin_likely: bool
    source: str


def parse_args() -> argparse.Namespace:
    """Parse CLI arguments."""

    parser = argparse.ArgumentParser(
        description="Generate an Audiobookshelf API inventory from ApiRouter.js.",
    )
    source = parser.add_mutually_exclusive_group(required=True)
    source.add_argument(
        "--source-dir",
        type=Path,
        help="Path to an Audiobookshelf source checkout.",
    )
    source.add_argument(
        "--router-file",
        type=Path,
        help="Path to an ApiRouter.js file.",
    )
    parser.add_argument(
        "--source-ref",
        default="unknown",
        help="Audiobookshelf source commit, tag, branch, or URL used for this inventory.",
    )
    parser.add_argument(
        "--output-json",
        type=Path,
        default=Path("docs/api-inventory/generated/abs-api-inventory.json"),
        help="Path for generated JSON inventory.",
    )
    parser.add_argument(
        "--output-md",
        type=Path,
        default=Path("docs/api-inventory/generated/abs-api-inventory.md"),
        help="Path for generated Markdown inventory.",
    )
    return parser.parse_args()


def resolve_router_file(source_dir: Path | None, router_file: Path | None) -> Path:
    """Resolve the ApiRouter.js path from either a source directory or direct file."""

    if router_file is not None:
        return router_file
    if source_dir is None:
        raise ValueError("source_dir or router_file is required")
    return source_dir / ROUTER_RELATIVE_PATH


def find_route_calls(source: str) -> list[tuple[str, str]]:
    """Extract this.router.<method>(...) calls with balanced parentheses."""

    calls: list[tuple[str, str]] = []
    marker = "this.router."
    index = 0

    while True:
        start = source.find(marker, index)
        if start == -1:
            break

        method_start = start + len(marker)
        method_end = method_start
        while method_end < len(source) and (
            source[method_end].isalpha() or source[method_end] == "_"
        ):
            method_end += 1

        method = source[method_start:method_end]
        if method not in ROUTE_METHODS:
            index = method_end
            continue

        open_paren = source.find("(", method_end)
        if open_paren == -1:
            break

        close_paren = find_balanced_close(source, open_paren)
        if close_paren == -1:
            raise ValueError(f"unbalanced route call for this.router.{method}")

        calls.append((method.upper(), source[open_paren + 1 : close_paren]))
        index = close_paren + 1

    return calls


def find_balanced_close(source: str, open_paren: int) -> int:
    """Find the closing parenthesis matching open_paren."""

    depth = 0
    quote: str | None = None
    escape = False

    for index in range(open_paren, len(source)):
        char = source[index]

        if quote is not None:
            if escape:
                escape = False
            elif char == "\\":
                escape = True
            elif char == quote:
                quote = None
            continue

        if char in {"'", '"', "`"}:
            quote = char
            continue
        if char == "(":
            depth += 1
        elif char == ")":
            depth -= 1
            if depth == 0:
                return index

    return -1


def split_top_level_args(arguments: str) -> list[str]:
    """Split function arguments on top-level commas."""

    parts: list[str] = []
    start = 0
    depth = 0
    quote: str | None = None
    escape = False

    for index, char in enumerate(arguments):
        if quote is not None:
            if escape:
                escape = False
            elif char == "\\":
                escape = True
            elif char == quote:
                quote = None
            continue

        if char in {"'", '"', "`"}:
            quote = char
            continue
        if char in "([{":
            depth += 1
        elif char in ")]}":
            depth -= 1
        elif char == "," and depth == 0:
            parts.append(arguments[start:index].strip())
            start = index + 1

    tail = arguments[start:].strip()
    if tail:
        parts.append(tail)
    return parts


def parse_path(path_expression: str) -> tuple[str, str]:
    """Parse a route path expression into a display path and path kind."""

    expression = path_expression.strip()
    if len(expression) >= 2 and expression[0] in {"'", '"', "`"}:
        quote = expression[0]
        end = expression.rfind(quote)
        return expression[1:end], "literal"
    if expression.startswith("/"):
        return expression, "regex"
    return expression, "expression"


def extract_controller_bindings(arguments: Sequence[str]) -> list[tuple[str, str]]:
    """Return controller method bindings in argument order."""

    bindings: list[tuple[str, str]] = []
    pattern = re.compile(r"\b([A-Za-z0-9_]+Controller)\.([A-Za-z0-9_]+)\.bind\(this\)")
    for argument in arguments:
        bindings.extend(pattern.findall(argument))
    return bindings


def route_group(path: str) -> str:
    """Classify a path into its top-level API group."""

    if path.startswith("/^\\/libraries") or "libraries" in path and path.startswith("/^"):
        return "libraries"
    if not path.startswith("/"):
        return "unknown"
    stripped = path.lstrip("/")
    if not stripped:
        return "root"
    return stripped.split("/", 1)[0].split(":", 1)[0] or "unknown"


def middleware_names(arguments: Sequence[str], handler: tuple[str, str] | None) -> list[str]:
    """Collect middleware-ish argument names, excluding the final handler binding."""

    names: list[str] = []
    for argument in arguments[1:]:
        cleaned = argument.strip()
        if handler and cleaned == f"{handler[0]}.{handler[1]}.bind(this)":
            continue
        if ".bind(this)" in cleaned:
            names.append(cleaned.replace(".bind(this)", ""))
        elif cleaned:
            names.append(cleaned)
    return names


def infer_admin_likely(middleware: Sequence[str]) -> bool:
    """Infer whether a route likely requires admin-like access from middleware names."""

    return any("adminMiddleware" in item or "ApiKeyController.middleware" in item for item in middleware)


def build_inventory(router_file: Path, source_ref: str) -> dict[str, object]:
    """Build the full inventory document."""

    source = router_file.read_text(encoding="utf-8")
    routes: list[RouteBinding] = []

    for method, raw_arguments in find_route_calls(source):
        arguments = split_top_level_args(raw_arguments)
        if not arguments:
            continue

        path, path_kind = parse_path(arguments[0])
        bindings = extract_controller_bindings(arguments[1:])
        handler = bindings[-1] if bindings else ("", "")
        middleware = middleware_names(arguments, handler if handler != ("", "") else None)

        routes.append(
            RouteBinding(
                method=method,
                path=path,
                path_kind=path_kind,
                group=route_group(path),
                handler_controller=handler[0],
                handler_method=handler[1],
                middleware=middleware,
                mutates=method.lower() in MUTATING_METHODS,
                admin_likely=infer_admin_likely(middleware),
                source=str(ROUTER_RELATIVE_PATH),
            ),
        )

    routes.sort(key=lambda route: (route.group, route.path, route.method))
    by_method = Counter(route.method for route in routes)
    by_group = Counter(route.group for route in routes)

    return {
        "source": {
            "source_ref": source_ref,
            "source_path": str(ROUTER_RELATIVE_PATH),
        },
        "summary": {
            "total_routes": len(routes),
            "mutating_routes": sum(1 for route in routes if route.mutates),
            "read_only_routes": sum(1 for route in routes if not route.mutates),
            "by_method": dict(sorted(by_method.items())),
            "by_group": dict(sorted(by_group.items())),
        },
        "routes": [asdict(route) for route in routes],
    }


def write_json(path: Path, inventory: dict[str, object]) -> None:
    """Write stable pretty JSON."""

    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(inventory, indent=2, sort_keys=True) + "\n", encoding="utf-8")


def write_markdown(path: Path, inventory: dict[str, object]) -> None:
    """Write a compact Markdown summary of the inventory."""

    source = inventory["source"]
    summary = inventory["summary"]
    routes = inventory["routes"]
    if not isinstance(source, dict) or not isinstance(summary, dict) or not isinstance(routes, list):
        raise TypeError("unexpected inventory shape")

    lines = [
        "# Audiobookshelf API Inventory",
        "",
        f"- Source ref: `{source['source_ref']}`",
        f"- Source path: `{source['source_path']}`",
        f"- Total routes: `{summary['total_routes']}`",
        f"- Read-only routes: `{summary['read_only_routes']}`",
        f"- Mutating routes: `{summary['mutating_routes']}`",
        "",
        "## Routes",
        "",
        "| Method | Path | Handler | Middleware | Mutates |",
        "| --- | --- | --- | --- | --- |",
    ]

    for route in routes:
        if not isinstance(route, dict):
            raise TypeError("unexpected route shape")
        handler = ""
        if route["handler_controller"] and route["handler_method"]:
            handler = f"{route['handler_controller']}.{route['handler_method']}"
        middleware = ", ".join(route["middleware"]) if route["middleware"] else ""
        lines.append(
            f"| `{route['method']}` | `{route['path']}` | `{handler}` | "
            f"{markdown_cell(middleware)} | `{str(route['mutates']).lower()}` |",
        )

    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text("\n".join(lines) + "\n", encoding="utf-8")


def markdown_cell(value: str) -> str:
    """Escape a Markdown table cell."""

    if not value:
        return ""
    return f"`{value.replace('|', '\\|')}`"


def main() -> None:
    """Run the inventory generator."""

    args = parse_args()
    router_file = resolve_router_file(args.source_dir, args.router_file)
    if not router_file.is_file():
        raise SystemExit(f"ApiRouter.js not found: {router_file}")

    inventory = build_inventory(router_file, args.source_ref)
    write_json(args.output_json, inventory)
    write_markdown(args.output_md, inventory)

    summary = inventory["summary"]
    if not isinstance(summary, dict):
        raise TypeError("unexpected inventory summary")
    print(
        "Generated ABS API inventory: "
        f"{summary['total_routes']} routes "
        f"({summary['read_only_routes']} read-only, {summary['mutating_routes']} mutating)",
    )
    print(f"JSON: {args.output_json}")
    print(f"Markdown: {args.output_md}")


if __name__ == "__main__":
    main()
