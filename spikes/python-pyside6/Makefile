.DEFAULT_GOAL := help
.PHONY: help install run lint format test check selftest dist clean

# Works anywhere GNU make and uv are available (Linux, macOS, Git Bash / MSYS on Windows).
UV ?= uv

help: ## Show this help
	@grep -E '^[a-z-]+:.*## ' $(MAKEFILE_LIST) | awk -F':.*## ' '{printf "  %-10s %s\n", $$1, $$2}'

install: ## Create the virtualenv and install dependencies
	$(UV) sync

run: ## Launch the app
	$(UV) run lumen

lint: ## Ruff lint + format check
	$(UV) run ruff check .
	$(UV) run ruff format --check .

format: ## Auto-fix lint issues and format
	$(UV) run ruff check --fix .
	$(UV) run ruff format .

test: ## Run the tests (headless)
	$(UV) run pytest

check: lint test ## Lint + test

selftest: ## Boot the app headless and drive it through its API
	QT_QPA_PLATFORM=offscreen $(UV) run lumen --self-test

dist: ## Build a standalone app into dist/ and smoke-test it
	$(UV) run python scripts/build.py --self-test

clean: ## Remove build artefacts and caches
	rm -rf build dist .pytest_cache .ruff_cache
