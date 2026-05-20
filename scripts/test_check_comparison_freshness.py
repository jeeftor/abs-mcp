"""Tests for README comparison freshness warnings."""

from __future__ import annotations

import datetime as dt
import tempfile
import unittest
from pathlib import Path

from check_comparison_freshness import check_freshness


class ComparisonFreshnessTest(unittest.TestCase):
    """Test the non-blocking README comparison freshness check."""

    def test_fresh_timestamp_returns_no_warning(self) -> None:
        """A recent comparison timestamp should not warn."""
        with tempfile.TemporaryDirectory() as temp_dir:
            readme = Path(temp_dir) / "README.md"
            readme.write_text(
                "## AI Generated Comparison - Last updated 2026-05-20\n",
                encoding="utf-8",
            )

            warning = check_freshness(readme, 7, dt.date(2026, 5, 27))

        self.assertEqual(warning, "")

    def test_stale_timestamp_warns(self) -> None:
        """An old comparison timestamp should suggest rerunning the skill."""
        with tempfile.TemporaryDirectory() as temp_dir:
            readme = Path(temp_dir) / "README.md"
            readme.write_text(
                "## AI Generated Comparison - Last updated 2026-05-20\n",
                encoding="utf-8",
            )

            warning = check_freshness(readme, 7, dt.date(2026, 5, 28))

        self.assertIn("8 days old", warning)
        self.assertIn("$abs-mcp-comparison", warning)

    def test_missing_timestamp_warns(self) -> None:
        """A README without the marker should suggest rerunning the skill."""
        with tempfile.TemporaryDirectory() as temp_dir:
            readme = Path(temp_dir) / "README.md"
            readme.write_text("# Project\n", encoding="utf-8")

            warning = check_freshness(readme, 7, dt.date(2026, 5, 20))

        self.assertIn("no AI Generated Comparison timestamp", warning)
        self.assertIn("$abs-mcp-comparison", warning)


if __name__ == "__main__":
    unittest.main()
