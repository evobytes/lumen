# lumen - the only entry point. CI workflows call these targets and nothing else.
# Logic lives in scripts/; this file wires targets to them.

SHELL := bash
.SHELLFLAGS := -euo pipefail -c
.DEFAULT_GOAL := help
.DELETE_ON_ERROR:

MOD     := $(shell go list -m)
BIN     := lumen
TOOLBIN := $(CURDIR)/.tools/bin
export PATH := $(TOOLBIN):$(PATH)

# Version info is computed ONCE per make run so every artifact shares a stamp.
VERINFO         := $(shell scripts/version)
VERSION         := $(patsubst VERSION=%,%,$(filter VERSION=%,$(VERINFO)))
GIT_HASH        := $(patsubst GIT_HASH=%,%,$(filter GIT_HASH=%,$(VERINFO)))
BUILD_STAMP     := $(patsubst BUILD_STAMP=%,%,$(filter BUILD_STAMP=%,$(VERINFO)))
ARTIFACT_SUFFIX := $(patsubst ARTIFACT_SUFFIX=%,%,$(filter ARTIFACT_SUFFIX=%,$(VERINFO)))
export BUILD_STAMP

# Only our own Go source is formatted: never the repo-local caches (.cache, .tools), reference clones
# (.ref), build output (dist) or the archived spikes (own modules).
GOSRC := cmd internal scripts

HOST_OS   := $(shell go env GOOS)
HOST_ARCH := $(shell go env GOARCH)

# Platforms `make vet` and `make lint` check. Windows is always checked (it is CGO-free, so
# it compiles on any host); the host platform is added when it differs (Linux and macOS need
# CGO and their GUI libraries, so they can only be checked natively). Each CI job passes
# CHECK_OS=host: the other jobs cover the other platforms.
CHECK_OS ?= windows $(filter-out windows,$(HOST_OS))
ifeq ($(CHECK_OS),host)
override CHECK_OS := $(HOST_OS)
endif

.PHONY: help
help: ## list targets
	@grep -hE '^[a-zA-Z0-9_.-]+:.*## ' $(MAKEFILE_LIST) | sort | awk -F':.*## ' '{printf "  %-14s %s\n", $$1, $$2}'

.PHONY: version
version: ## print version, git hash, build stamp and artifact suffix
	@echo "version=$(VERSION) hash=$(GIT_HASH) stamp=$(BUILD_STAMP)"
	@echo "suffix=$(ARTIFACT_SUFFIX)"
	@echo "example: dist/$(BIN)_$(VERSION)_windows-amd64_$(ARTIFACT_SUFFIX).zip"

.PHONY: bootstrap
bootstrap: ## install pinned dev tools into .tools/bin
	scripts/bootstrap

# ---- develop ---------------------------------------------------------------

.PHONY: dev
dev: ## run the app on this host with dev/config.toml on port 18765 (Ctrl-C stops it)
	go run ./cmd/lumen -config dev/config.toml

.PHONY: selftest
selftest: ## build the console binary and drive the real app: placement on every monitor, media probe; report in dist/selftest/
	scripts/selftest

# ---- check -----------------------------------------------------------------

.PHONY: fmt
fmt: ## format code (goimports + gofumpt)
	goimports -local $(MOD) -w $(GOSRC)
	gofumpt -w $(GOSRC)

.PHONY: fmt-check
fmt-check: ## fail if code is not formatted
	@imp="$$(goimports -local $(MOD) -l $(GOSRC))" || exit 1; \
	fum="$$(gofumpt -l $(GOSRC))" || exit 1; \
	out="$$imp$$fum"; \
	if [[ -n "$$out" ]]; then echo "not formatted (run 'make fmt'):"; echo "$$out"; exit 1; fi

.PHONY: vet
vet: ## go vet for windows and the host platform (CHECK_OS=host: only the host)
	@set -e; for os in $(CHECK_OS); do \
	  cgo=1; [ "$$os" = windows ] && cgo=0; \
	  echo "GOOS=$$os CGO_ENABLED=$$cgo go vet ./..."; GOOS=$$os CGO_ENABLED=$$cgo go vet ./...; done

.PHONY: lint
lint: ## golangci-lint for windows and the host platform (CHECK_OS=host: only the host)
	@set -e; for os in $(CHECK_OS); do \
	  cgo=1; [ "$$os" = windows ] && cgo=0; \
	  echo "GOOS=$$os CGO_ENABLED=$$cgo golangci-lint run ./..."; GOOS=$$os CGO_ENABLED=$$cgo golangci-lint run ./...; done

.PHONY: test
test: ## unit tests with the race detector (needs cgo: gcc on Windows); shuffled
	go test -race -shuffle=on -count=1 ./...

.PHONY: test-windows
test-windows: ## unit tests without the race detector (what make ci runs on Windows)
	go test -shuffle=on -count=1 ./...

.PHONY: cover
cover: ## coverage report to dist/coverage.html
	@mkdir -p dist
	go test -count=1 -coverprofile=dist/coverage.out ./...
	go tool cover -html=dist/coverage.out -o dist/coverage.html
	go tool cover -func=dist/coverage.out | tail -1

.PHONY: vuln
vuln: ## govulncheck
	govulncheck ./...

.PHONY: docs
docs: ## regenerate generated docs (config reference, API reference, make targets, skills index, packages, versions, notices)
	go run ./scripts/docgen generate >/dev/null

.PHONY: docs-check
docs-check: ## fail if generated docs are stale or docs reference things that no longer exist
	go run ./scripts/docgen check

.PHONY: docs-bundle
docs-bundle: ## regenerate and check the docs, then write a reproducible docs zip to dist/
	scripts/build-docs

.PHONY: hooks
hooks: ## install the git pre-commit hook (regenerates and stages docs); run once per clone
	scripts/install-hooks

.PHONY: exec-check
exec-check: ## fail if a tracked script is not stored as executable in git
	scripts/check-exec

.PHONY: tidy-check
tidy-check: ## fail if go.mod/go.sum are not tidy
	scripts/check-tidy

# The race detector needs cgo, which a Windows dev host (and runner) lacks by default:
# there the gate runs the suite without -race, and the Linux CI job keeps -race.
ifeq ($(HOST_OS),windows)
CI_TEST := test-windows
else
CI_TEST := test
endif

.PHONY: ci
ci: fmt-check vet lint exec-check tidy-check docs-check $(CI_TEST) vuln build ## the full gate (fmt, vet, lint, script modes, tidy, docs, tests, vuln, build); identical on every CI job

# ---- build and package ------------------------------------------------------

.PHONY: build
build: ## host binary to dist/bin/, plus the Windows binary
	scripts/build-binary $(HOST_OS) $(HOST_ARCH)
	@if [ "$(HOST_OS)" != windows ]; then scripts/build-binary windows amd64; fi

.PHONY: deb
deb: ## .deb for the host architecture in dist/ (Linux only; needs libgtk-4-dev and libwebkitgtk-6.0-dev)
	scripts/build-deb

.PHONY: win-zip
win-zip: ## portable Windows zip (exe + example config + licences) in dist/; cross-builds from any host
	scripts/build-windows-zip amd64

.PHONY: installer
installer: ## Windows installer (Inno Setup, needs ISCC.exe on Windows) in dist/
	scripts/build-windows-installer amd64

.PHONY: macos-app
macos-app: ## Lumen.app zip in dist/ (macOS only; unsigned)
	scripts/build-macos-app

.PHONY: packages
packages: ## all packages the host can build: Linux deb + Windows zip; on Windows, zip + installer; on macOS, the app zip
	@case "$(HOST_OS)" in \
	  linux)   scripts/build-deb && scripts/build-windows-zip amd64 ;; \
	  windows) scripts/build-windows-zip amd64 && scripts/build-windows-installer amd64 ;; \
	  darwin)  scripts/build-macos-app ;; \
	esac

.PHONY: release-dry
release-dry: ci packages docs-bundle ## everything a release does, minus publishing

.PHONY: release-check
release-check: ## verify TAG=vX.X.X matches VERSION and the tree is clean
	scripts/release-check "$(TAG)"

.PHONY: clean
clean: ## remove dist/
	rm -rf dist
