#!/usr/bin/env python3
"""Validate repository MCP Registry metadata invariants."""

from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path
from typing import Any


def main() -> int:
    """Validate server.json for release-critical registry rules."""
    parser = argparse.ArgumentParser()
    parser.add_argument(
        "path",
        nargs="?",
        default=Path("server.json"),
        type=Path,
        help="Path to server.json.",
    )
    args = parser.parse_args()

    try:
        data = json.loads(args.path.read_text(encoding="utf-8"))
    except OSError as err:
        print(f"{args.path}: cannot read file: {err}", file=sys.stderr)
        return 1
    except json.JSONDecodeError as err:
        print(f"{args.path}: invalid JSON: {err}", file=sys.stderr)
        return 1

    failures = validate_metadata(data)
    if failures:
        for failure in failures:
            print(f"{args.path}: {failure}", file=sys.stderr)
        return 1
    return 0


def validate_metadata(data: dict[str, Any]) -> list[str]:
    """Return validation failures for MCP registry metadata."""
    failures: list[str] = []
    version = data.get("version")
    if not isinstance(version, str) or not version:
        failures.append("top-level version must be a non-empty string")

    packages = data.get("packages")
    if not isinstance(packages, list) or not packages:
        failures.append("packages must be a non-empty array")
        return failures

    for index, package in enumerate(packages):
        if not isinstance(package, dict):
            failures.append(f"packages[{index}] must be an object")
            continue
        registry_type = package.get("registryType")
        identifier = package.get("identifier")
        if not isinstance(identifier, str) or not identifier:
            failures.append(f"packages[{index}].identifier must be a non-empty string")
            continue
        if registry_type == "oci":
            failures.extend(validate_oci_package(index, package, identifier, version))

    return failures


def validate_oci_package(
    index: int,
    package: dict[str, Any],
    identifier: str,
    version: object,
) -> list[str]:
    """Return validation failures for one OCI package entry."""
    failures: list[str] = []
    if "version" in package:
        failures.append(
            f"packages[{index}].version must be omitted for OCI packages; "
            "put the version in packages[].identifier instead"
        )

    image, separator, tag = identifier.rpartition(":")
    if not separator or not image or not tag or "/" not in image:
        failures.append(f"packages[{index}].identifier must include an OCI image tag")
    elif isinstance(version, str) and tag != version:
        failures.append(
            f"packages[{index}].identifier tag {tag!r} must match top-level version {version!r}"
        )
    return failures


if __name__ == "__main__":
    raise SystemExit(main())
