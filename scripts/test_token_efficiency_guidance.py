"""Tests for token-use efficiency guidance and preview output shape."""

from __future__ import annotations

import re
import unittest
from pathlib import Path


REPO_ROOT = Path(__file__).resolve().parents[1]


class TokenEfficiencyGuidanceTest(unittest.TestCase):
    """Validate token-efficiency guardrails for MCP feature work."""

    def test_agents_guidance_requires_token_efficiency(self) -> None:
        """Agent guidance should make token-use efficiency a default design goal."""
        agents = (REPO_ROOT / "AGENTS.md").read_text(encoding="utf-8")

        self.assertIn("Always target token-use efficiency", agents)
        self.assertIn("bounded result sets", agents)
        self.assertIn("compact summaries", agents)

    def test_preview_tool_uses_compact_structured_output(self) -> None:
        """The device-send preview should expose summaries, not raw ABS payloads."""
        tools_go = (REPO_ROOT / "internal" / "mcpserver" / "tools.go").read_text(
            encoding="utf-8"
        )
        match = re.search(
            r"type PreviewEbookDeviceSendOutput struct \{(?P<body>.*?)\n\}",
            tools_go,
            flags=re.S,
        )

        self.assertIsNotNone(match, "PreviewEbookDeviceSendOutput should exist")
        body = match.group("body")
        self.assertIn("[]LibraryItemSummary", body)
        self.assertIn("[]EReaderDeviceSummary", body)
        self.assertNotIn("abs.JSONValue", body)

    def test_preview_docs_describe_compact_bounded_flow(self) -> None:
        """Docs should tell callers the preview is compact and bounded."""
        docs = (REPO_ROOT / "docs" / "tools.md").read_text(encoding="utf-8")
        readme = (REPO_ROOT / "README.md").read_text(encoding="utf-8")

        self.assertRegex(docs, r"abs_preview_ebook_device_send.*Compact ebook")
        self.assertRegex(docs, r"abs_preview_ebook_device_send.*maxCandidates")
        self.assertIn("returns compact ebook candidates", readme)


if __name__ == "__main__":
    unittest.main()
