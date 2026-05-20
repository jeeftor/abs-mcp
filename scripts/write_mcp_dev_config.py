#!/usr/bin/env python3
"""Write a local MCP client config for the Docker Audiobookshelf fixture."""

from __future__ import annotations

import argparse
import json
import os
from pathlib import Path
from typing import Mapping


DEFAULT_SERVER_NAME = "abs-mcp"
DEFAULT_BASE_URL_KEY = "ABS_PLAIN_URL"


def parse_args() -> argparse.Namespace:
    """Parse CLI arguments."""

    parser = argparse.ArgumentParser(
        description="Write a local MCP client config for abs-mcp development.",
    )
    parser.add_argument(
        "--env-file",
        type=Path,
        default=Path("test/abs/.env.testing"),
        help="Fixture environment file containing ABS_PLAIN_URL and ABS_TOKEN.",
    )
    parser.add_argument(
        "--output",
        type=Path,
        default=Path(".mcp.dev.json"),
        help="Output JSON config path.",
    )
    parser.add_argument(
        "--server-name",
        default=DEFAULT_SERVER_NAME,
        help="MCP server name to use in the generated config.",
    )
    parser.add_argument(
        "--base-url-key",
        default=DEFAULT_BASE_URL_KEY,
        help="Environment key to use for ABS_BASE_URL.",
    )
    parser.add_argument(
        "--read-write",
        action="store_true",
        help="Allow mutating MCP tools by setting ABS_READ_ONLY=false.",
    )
    return parser.parse_args()


def read_dotenv(path: Path) -> dict[str, str]:
    """Read simple KEY=VALUE lines from a dotenv file."""

    values: dict[str, str] = {}
    for raw_line in path.read_text(encoding="utf-8").splitlines():
        line = raw_line.strip()
        if not line or line.startswith("#"):
            continue
        key, separator, value = line.partition("=")
        if not separator:
            continue
        values[key.strip()] = strip_quotes(value.strip())
    return values


def strip_quotes(value: str) -> str:
    """Remove one matching layer of dotenv-style quotes."""

    if len(value) >= 2 and value[0] == value[-1] and value[0] in {"'", '"'}:
        return value[1:-1]
    return value


def require_value(values: Mapping[str, str], key: str) -> str:
    """Return a required dotenv value or raise a clear error."""

    value = values.get(key, "")
    if not value:
        raise ValueError(f"missing required value {key}")
    return value


def build_config(
    repo_root: Path,
    values: Mapping[str, str],
    *,
    server_name: str = DEFAULT_SERVER_NAME,
    base_url_key: str = DEFAULT_BASE_URL_KEY,
    read_only: bool = True,
) -> dict[str, object]:
    """Build an MCP client config object."""

    return {
        "mcpServers": {
            server_name: {
                "command": str(repo_root / "bin" / "abs-mcp"),
                "args": [],
                "env": {
                    "ABS_BASE_URL": require_value(values, base_url_key),
                    "ABS_API_KEY": require_value(values, "ABS_TOKEN"),
                    "ABS_READ_ONLY": "true" if read_only else "false",
                    "ABS_TIMEOUT": "30s",
                    "ABS_FIXTURE_DIR": str(repo_root / "test" / "abs"),
                },
            },
        },
    }


def write_json_private(path: Path, value: Mapping[str, object]) -> None:
    """Write JSON with owner-only permissions because it contains a fixture token."""

    path.parent.mkdir(parents=True, exist_ok=True)
    flags = os.O_WRONLY | os.O_CREAT | os.O_TRUNC
    file_descriptor = os.open(path, flags, 0o600)
    with os.fdopen(file_descriptor, "w", encoding="utf-8") as handle:
        json.dump(value, handle, indent=2)
        handle.write("\n")
    path.chmod(0o600)


def main() -> int:
    """Write the requested config and print a redacted summary."""

    args = parse_args()
    repo_root = Path(__file__).resolve().parents[1]
    env_file = resolve_from_repo(repo_root, args.env_file)
    output = resolve_from_repo(repo_root, args.output)
    values = read_dotenv(env_file)
    read_only = not args.read_write
    config = build_config(
        repo_root,
        values,
        server_name=args.server_name,
        base_url_key=args.base_url_key,
        read_only=read_only,
    )

    write_json_private(output, config)
    server = config["mcpServers"][args.server_name]  # type: ignore[index]
    server_env = server["env"]  # type: ignore[index]
    print(f"Wrote MCP dev config: {output}")
    print(f"Server: {args.server_name}")
    print(f"Command: {server['command']}")  # type: ignore[index]
    print(f"ABS_BASE_URL: {server_env['ABS_BASE_URL']}")  # type: ignore[index]
    print(f"ABS_READ_ONLY: {server_env['ABS_READ_ONLY']}")  # type: ignore[index]
    print("ABS_API_KEY: <redacted>")
    return 0


def resolve_from_repo(repo_root: Path, path: Path) -> Path:
    """Resolve relative paths from the repository root."""

    if path.is_absolute():
        return path
    return repo_root / path


if __name__ == "__main__":
    raise SystemExit(main())
