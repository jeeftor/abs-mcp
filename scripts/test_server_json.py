"""Tests for MCP Registry metadata validation."""

from __future__ import annotations

import subprocess
import sys
import tempfile
import unittest
from pathlib import Path

SCRIPT_DIR = Path(__file__).resolve().parent
sys.path.insert(0, str(SCRIPT_DIR))

import validate_server_json


class ServerJSONValidationTests(unittest.TestCase):
    """Verify release-critical server.json invariants."""

    def test_valid_oci_package_accepts_versioned_identifier(self) -> None:
        failures = validate_server_json.validate_metadata(
            {
                "version": "0.2.1",
                "packages": [
                    {
                        "registryType": "oci",
                        "identifier": "ghcr.io/jeeftor/abs-mcp:0.2.1",
                    }
                ],
            }
        )

        self.assertEqual(failures, [])

    def test_oci_package_rejects_package_version_field(self) -> None:
        failures = validate_server_json.validate_metadata(
            {
                "version": "0.2.1",
                "packages": [
                    {
                        "registryType": "oci",
                        "identifier": "ghcr.io/jeeftor/abs-mcp:0.2.1",
                        "version": "0.2.1",
                    }
                ],
            }
        )

        self.assertIn("packages[0].version must be omitted", failures[0])

    def test_oci_package_tag_must_match_top_level_version(self) -> None:
        failures = validate_server_json.validate_metadata(
            {
                "version": "0.2.1",
                "packages": [
                    {
                        "registryType": "oci",
                        "identifier": "ghcr.io/jeeftor/abs-mcp:0.2.0",
                    }
                ],
            }
        )

        self.assertIn("must match top-level version", failures[0])

    def test_cli_reports_invalid_metadata(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            path = Path(temp_dir) / "server.json"
            path.write_text(
                """
                {
                  "version": "0.2.1",
                  "packages": [
                    {
                      "registryType": "oci",
                      "identifier": "ghcr.io/jeeftor/abs-mcp:0.2.1",
                      "version": "0.2.1"
                    }
                  ]
                }
                """,
                encoding="utf-8",
            )

            result = subprocess.run(
                [sys.executable, str(SCRIPT_DIR / "validate_server_json.py"), str(path)],
                check=False,
                capture_output=True,
                text=True,
            )

        self.assertEqual(result.returncode, 1)
        self.assertIn("packages[0].version must be omitted", result.stderr)


if __name__ == "__main__":
    unittest.main()
