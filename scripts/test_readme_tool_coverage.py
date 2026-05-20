"""Tests for README MCP tool coverage."""

from __future__ import annotations

import re
import unittest
from pathlib import Path


REPO_ROOT = Path(__file__).resolve().parents[1]
TOOL_NAME_RE = re.compile(r'Name:\s+"(abs_[a-z_]+)"')
README_TOOL_RE = re.compile(r"`(abs_[a-z_]+)`")


class ReadmeToolCoverageTest(unittest.TestCase):
    """Validate that README documents all registered MCP tools."""

    def test_readme_lists_all_registered_tools(self) -> None:
        """Every registered tool name should be present in README.md."""
        tools_go = REPO_ROOT / "internal" / "mcpserver" / "tools.go"
        readme = REPO_ROOT / "README.md"

        registered = set(TOOL_NAME_RE.findall(tools_go.read_text(encoding="utf-8")))
        documented = set(README_TOOL_RE.findall(readme.read_text(encoding="utf-8")))

        self.assertTrue(registered, "expected registered MCP tools")
        self.assertEqual(set(), registered - documented)


if __name__ == "__main__":
    unittest.main()
