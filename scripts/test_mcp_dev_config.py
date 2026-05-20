"""Tests for the local MCP development config writer."""

from __future__ import annotations

import json
import stat
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path

SCRIPT_DIR = Path(__file__).resolve().parent
REPO_ROOT = SCRIPT_DIR.parent
sys.path.insert(0, str(SCRIPT_DIR))

import write_mcp_dev_config


class MCPDevConfigTests(unittest.TestCase):
    """Verify generated MCP config shape and token handling."""

    def test_build_config_uses_fixture_values_and_binary_command(self) -> None:
        config = write_mcp_dev_config.build_config(
            REPO_ROOT,
            {
                "ABS_PLAIN_URL": "http://localhost:13388",
                "ABS_TOKEN": "secret-token",
            },
        )

        server = config["mcpServers"]["abs-mcp"]  # type: ignore[index]
        self.assertEqual(server["command"], str(REPO_ROOT / "bin" / "abs-mcp"))
        self.assertEqual(server["args"], [])
        self.assertEqual(
            server["env"],
            {
                "ABS_BASE_URL": "http://localhost:13388",
                "ABS_API_KEY": "secret-token",
                "ABS_READ_ONLY": "true",
                "ABS_TIMEOUT": "30s",
                "ABS_FIXTURE_DIR": str(REPO_ROOT / "test" / "abs"),
            },
        )

    def test_build_config_can_enable_read_write_tools(self) -> None:
        config = write_mcp_dev_config.build_config(
            REPO_ROOT,
            {
                "ABS_PLAIN_URL": "http://localhost:13388",
                "ABS_TOKEN": "secret-token",
            },
            read_only=False,
        )

        server = config["mcpServers"]["abs-mcp"]  # type: ignore[index]
        self.assertEqual(server["env"]["ABS_READ_ONLY"], "false")

    def test_cli_writes_private_config_without_printing_token(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            temp_path = Path(temp_dir)
            env_file = temp_path / ".env.testing"
            output = temp_path / "mcp.json"
            env_file.write_text(
                "ABS_PLAIN_URL=http://localhost:13388\n"
                "ABS_TOKEN=secret-token\n",
                encoding="utf-8",
            )

            result = subprocess.run(
                [
                    sys.executable,
                    str(SCRIPT_DIR / "write_mcp_dev_config.py"),
                    "--env-file",
                    str(env_file),
                    "--output",
                    str(output),
                ],
                check=False,
                capture_output=True,
                text=True,
            )

            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertNotIn("secret-token", result.stdout)
            self.assertIn("ABS_API_KEY: <redacted>", result.stdout)

            written = json.loads(output.read_text(encoding="utf-8"))
            server = written["mcpServers"]["abs-mcp"]
            self.assertEqual(server["env"]["ABS_API_KEY"], "secret-token")
            permissions = stat.S_IMODE(output.stat().st_mode)
            self.assertEqual(permissions, 0o600)


if __name__ == "__main__":
    unittest.main()
