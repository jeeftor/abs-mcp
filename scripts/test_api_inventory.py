"""Tests for Audiobookshelf API inventory helper scripts."""

from __future__ import annotations

import json
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path

SCRIPT_DIR = Path(__file__).resolve().parent
sys.path.insert(0, str(SCRIPT_DIR))

import diff_abs_api_inventory
import generate_abs_api_inventory


class GenerateInventoryTests(unittest.TestCase):
    """Verify source route extraction for representative ABS routes."""

    def test_build_inventory_extracts_routes_and_mutation_state(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            router_file = Path(temp_dir) / "ApiRouter.js"
            router_file.write_text(
                """
                class ApiRouter {
                  init() {
                    this.router.get('/libraries/:id/items',
                      LibraryController.middleware.bind(this),
                      LibraryController.getLibraryItems.bind(this))
                    this.router.delete('/libraries/:id/issues',
                      adminMiddleware,
                      LibraryController.middleware.bind(this),
                      LibraryController.removeLibraryItemsWithIssues.bind(this))
                  }
                }
                """,
                encoding="utf-8",
            )

            inventory = generate_abs_api_inventory.build_inventory(router_file, "test-ref")

        self.assertEqual(inventory["source"]["source_ref"], "test-ref")
        self.assertEqual(inventory["summary"]["total_routes"], 2)
        self.assertEqual(inventory["summary"]["read_only_routes"], 1)
        self.assertEqual(inventory["summary"]["mutating_routes"], 1)

        routes = {
            (route["method"], route["path"]): route
            for route in inventory["routes"]
        }
        items_route = routes[("GET", "/libraries/:id/items")]
        self.assertFalse(items_route["mutates"])
        self.assertEqual(items_route["handler_controller"], "LibraryController")
        self.assertEqual(items_route["handler_method"], "getLibraryItems")

        issues_route = routes[("DELETE", "/libraries/:id/issues")]
        self.assertTrue(issues_route["mutates"])
        self.assertTrue(issues_route["admin_likely"])
        self.assertEqual(
            issues_route["handler_method"],
            "removeLibraryItemsWithIssues",
        )


class DiffInventoryTests(unittest.TestCase):
    """Verify route drift detection and failing checks."""

    def test_changed_fields_detects_mcp_relevant_route_changes(self) -> None:
        before = {
            "group": "libraries",
            "handler_controller": "LibraryController",
            "handler_method": "getLibraryItems",
            "middleware": ["LibraryController.middleware"],
            "mutates": False,
            "admin_likely": False,
        }
        after = before | {"handler_method": "getItems", "mutates": True}

        self.assertEqual(
            diff_abs_api_inventory.changed_fields(before, after),
            ["handler_method", "mutates"],
        )

    def test_cli_fail_on_change_exits_nonzero_for_route_drift(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            temp_path = Path(temp_dir)
            baseline = temp_path / "baseline.json"
            generated = temp_path / "generated.json"
            write_inventory(
                baseline,
                [
                    {
                        "method": "GET",
                        "path": "/libraries",
                        "group": "libraries",
                        "handler_controller": "LibraryController",
                        "handler_method": "getAll",
                        "middleware": [],
                        "mutates": False,
                        "admin_likely": False,
                    },
                ],
            )
            write_inventory(
                generated,
                [
                    {
                        "method": "GET",
                        "path": "/libraries",
                        "group": "libraries",
                        "handler_controller": "LibraryController",
                        "handler_method": "getAll",
                        "middleware": [],
                        "mutates": False,
                        "admin_likely": False,
                    },
                    {
                        "method": "POST",
                        "path": "/libraries/:id/scan",
                        "group": "libraries",
                        "handler_controller": "LibraryController",
                        "handler_method": "scan",
                        "middleware": ["LibraryController.middleware"],
                        "mutates": True,
                        "admin_likely": False,
                    },
                ],
            )

            result = subprocess.run(
                [
                    sys.executable,
                    str(SCRIPT_DIR / "diff_abs_api_inventory.py"),
                    "--baseline",
                    str(baseline),
                    "--generated",
                    str(generated),
                    "--fail-on-change",
                ],
                check=False,
                capture_output=True,
                text=True,
            )

        self.assertEqual(result.returncode, 1)
        self.assertIn("POST /libraries/:id/scan", result.stdout)
        self.assertIn("Total changes: 1", result.stdout)


def write_inventory(path: Path, routes: list[dict[str, object]]) -> None:
    """Write a minimal inventory file for diff tests."""

    path.write_text(json.dumps({"routes": routes}), encoding="utf-8")


if __name__ == "__main__":
    unittest.main()
