ABS_COMPOSE = docker compose -f test/abs/docker-compose.yml
ABS_SOURCE_REF ?= local
UNIT_TEST_PKGS = ./...
PYTHON ?= python3
IMAGE ?= abs-mcp:dev

.PHONY: help build docker-build dev mcp-dev-config mcp-dev-config-read-write test test-unit test-go test-scripts abs-test-integration abs-dev-seed abs-dev-init abs-dev-configure abs-dev-up abs-dev-down abs-dev-reset abs-dev-reset-all abs-dev-scan abs-dev-reset-scan abs-ci-smoke abs-dev-capture-baseline abs-dev-restore-baseline abs-dev-wait abs-dev-ps abs-dev-config abs-api-inventory abs-api-inventory-from-router abs-api-inventory-diff abs-api-inventory-check

help:
	@echo "Available targets:"
	@echo ""
	@echo "  Development:"
	@printf "    %-26s %s\n" "dev" "Build server, start/reset/scan ABS, and write .mcp.dev.json"
	@printf "    %-26s %s\n" "mcp-dev-config" "Write read-only local MCP config to .mcp.dev.json"
	@printf "    %-26s %s\n" "mcp-dev-config-read-write" "Write read-write local MCP config to .mcp.dev.readwrite.json"
	@echo ""
	@echo "  Testing:"
	@printf "    %-26s %s\n" "build" "Build the abs-mcp stdio server"
	@printf "    %-26s %s\n" "test" "Run unit tests"
	@printf "    %-26s %s\n" "test-unit" "Run unit tests"
	@printf "    %-26s %s\n" "test-scripts" "Run Python script tests"
	@printf "    %-26s %s\n" "docker-build" "Build local abs-mcp Docker image"
	@printf "    %-26s %s\n" "abs-test-integration" "Run Docker-backed ABS integration tests"
	@echo ""
	@echo "  ABS fixture:"
	@printf "    %-26s %s\n" "abs-dev-seed" "Download public-domain ABS test media"
	@printf "    %-26s %s\n" "abs-dev-init" "Reset ABS and start with empty libraries for setup"
	@printf "    %-26s %s\n" "abs-dev-configure" "Configure empty ABS test servers through the API"
	@printf "    %-26s %s\n" "abs-dev-up" "Start local Audiobookshelf test servers"
	@printf "    %-26s %s\n" "abs-dev-wait" "Start ABS and wait until services respond"
	@printf "    %-26s %s\n" "abs-dev-down" "Stop local Audiobookshelf test servers"
	@printf "    %-26s %s\n" "abs-dev-reset" "Restore baseline, stage books, start ABS"
	@printf "    %-26s %s\n" "abs-dev-reset-all" "Reset ABS state and clear staged media"
	@printf "    %-26s %s\n" "abs-dev-scan" "Trigger scans for configured ABS libraries"
	@printf "    %-26s %s\n" "abs-dev-reset-scan" "Reset ABS, start it, and trigger scans"
	@printf "    %-26s %s\n" "abs-ci-smoke" "CI-style seed, restore baseline, and scan"
	@printf "    %-26s %s\n" "abs-dev-ps" "Show ABS fixture containers"
	@printf "    %-26s %s\n" "abs-dev-config" "Render Docker Compose config"
	@printf "    %-26s %s\n" "abs-api-inventory" "Generate API inventory from ABS_SOURCE_DIR"
	@printf "    %-26s %s\n" "abs-api-inventory-from-router" "Generate API inventory from ABS_ROUTER_FILE"
	@printf "    %-26s %s\n" "abs-api-inventory-diff" "Compare generated API inventory to baseline"
	@printf "    %-26s %s\n" "abs-api-inventory-check" "Fail when generated API inventory differs from baseline"

build:
	@mkdir -p bin
	@go build -o bin/abs-mcp ./cmd/abs-mcp

docker-build:
	@docker build -t $(IMAGE) .

dev: build abs-dev-reset-scan mcp-dev-config
	@echo ""
	@echo "ABS fixture is running at http://localhost:13388"
	@echo "MCP client config was written to .mcp.dev.json"
	@echo "Stop the fixture with: make abs-dev-down"

mcp-dev-config:
	@$(PYTHON) scripts/write_mcp_dev_config.py --output .mcp.dev.json

mcp-dev-config-read-write:
	@$(PYTHON) scripts/write_mcp_dev_config.py --output .mcp.dev.readwrite.json --read-write

test: test-unit

test-unit: test-go test-scripts

test-go:
	@go test $(UNIT_TEST_PKGS)

test-scripts:
	@$(PYTHON) -m unittest discover -s scripts -p 'test_*.py'

abs-test-integration: abs-dev-reset-scan
	@go test -tags=abs_integration ./test/abs/integration -count=1 -v

abs-dev-seed:
	@test/abs/scripts/seed-public-domain.sh

abs-dev-init:
	@test/abs/scripts/reset.sh --empty-runtime
	@$(ABS_COMPOSE) up -d --remove-orphans
	@test/abs/scripts/wait-for-abs.sh

abs-dev-configure:
	@test/abs/scripts/configure-from-api.sh

abs-dev-up:
	@$(ABS_COMPOSE) up -d --remove-orphans

abs-dev-down:
	@$(ABS_COMPOSE) down --remove-orphans

abs-dev-reset:
	@test/abs/scripts/reset.sh
	@test/abs/scripts/restore-baseline.sh
	@test/abs/scripts/wait-for-restored-config.sh
	@$(ABS_COMPOSE) up -d --remove-orphans
	@test/abs/scripts/wait-for-abs.sh

abs-dev-reset-all:
	@test/abs/scripts/reset.sh --clear-staging

abs-dev-scan:
	@test/abs/scripts/scan-libraries.sh

abs-dev-reset-scan: abs-dev-reset abs-dev-scan

abs-ci-smoke:
	@test/abs/scripts/seed-public-domain.sh
	@test/abs/scripts/reset.sh
	@test/abs/scripts/restore-baseline.sh
	@test/abs/scripts/wait-for-restored-config.sh
	@$(ABS_COMPOSE) up -d --remove-orphans
	@test/abs/scripts/wait-for-abs.sh
	@test/abs/scripts/scan-libraries.sh

abs-dev-capture-baseline:
	@test/abs/scripts/capture-baseline.sh

abs-dev-restore-baseline: abs-dev-reset

abs-dev-wait: abs-dev-up
	@test/abs/scripts/wait-for-abs.sh

abs-dev-ps:
	@$(ABS_COMPOSE) ps

abs-dev-config:
	@$(ABS_COMPOSE) config

abs-api-inventory:
	@if [ -z "$${ABS_SOURCE_DIR:-}" ]; then \
		echo "Set ABS_SOURCE_DIR to an Audiobookshelf source checkout."; \
		exit 2; \
	fi
	@scripts/generate_abs_api_inventory.py --source-dir "$$ABS_SOURCE_DIR" --source-ref "$(ABS_SOURCE_REF)"

abs-api-inventory-from-router:
	@if [ -z "$${ABS_ROUTER_FILE:-}" ]; then \
		echo "Set ABS_ROUTER_FILE to server/routers/ApiRouter.js."; \
		exit 2; \
	fi
	@scripts/generate_abs_api_inventory.py --router-file "$$ABS_ROUTER_FILE" --source-ref "$(ABS_SOURCE_REF)"

abs-api-inventory-diff:
	@scripts/diff_abs_api_inventory.py

abs-api-inventory-check:
	@scripts/diff_abs_api_inventory.py --fail-on-change
