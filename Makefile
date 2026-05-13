.PHONY: help bootstrap install generate fmt license vet lint lint-md \
        test test-generated test-race test-bench test-fuzz test-coverage \
        bench-baseline bench-regression \
        tidy check-tidy \
        check check-coverage check-vuln \
        build clean

# ─── Colors ──────────────────────────────────────────────────────
BLUE   := $(shell printf "\033[0;36m")
GREEN  := $(shell printf "\033[0;32m")
RED    := $(shell printf "\033[0;31m")
YELLOW := $(shell printf "\033[0;33m")
NC     := $(shell printf "\033[0m")

# ─── Go settings ─────────────────────────────────────────────────
GO      := go
FLAGS   ?=

# ─── Module directories ─────────────────────────────────────────
# Eidos is split into several Go modules coordinated by go.work.
# foreach_module iterates each one so the per-target work
# (`go test ./...`, `go vet ./...`, `go mod tidy`, …) runs in
# every module's scope rather than leaking across boundaries.
# Order matters for human readability: root first, then language
# adapters, then test harness.
MODULES := . ./backend/golang ./bridge/protogo ./cli ./eidostest ./frontend/golang ./frontend/protobuf ./reference

# ─── Paths ───────────────────────────────────────────────────────
BIN_DIR      := bin
COVERAGE_DIR := .eidos/coverage

# ─── Test tuning ─────────────────────────────────────────────────
# TEST_TIMEOUT applies to test, test-race, and test-bench. Override
# from the command line for slower runners or longer suites:
#   make test TEST_TIMEOUT=30m
TEST_CPU         ?= 4
TEST_COUNT       ?= 3
TEST_TIMEOUT     ?= 10m
TEST_RACE_COUNT  ?= 3
# BENCH_COUNT is the per-target run count for `make bench-baseline`
# and `make bench-regression`. benchstat compares distributions, so
# more samples mean tighter confidence bands. Default 6 keeps a CI
# run bounded; bump locally with BENCH_COUNT=12 for stable numbers.
BENCH_COUNT      ?= 6
# FUZZ_TIME is the per-target wall-clock budget for `go test -fuzz`.
# Default keeps a CI run bounded; bump locally with FUZZ_TIME=5m.
FUZZ_TIME        ?= 30s

# ─── License header ──────────────────────────────────────────────
GO_FILES := $(shell find . -type f -name '*.go' \
	! -path './.git/*' \
	! -name '*.gen.go' \
	! -name '*.gen_test.go')

# ─── Helper: run a command in each module ────────────────────────
# Usage: $(call foreach_module,go test ./...)
define foreach_module
	@for mod in $(MODULES); do \
		echo "$(BLUE)[$${mod}] $(1)$(NC)"; \
		(cd $$mod && $(1)) || exit 1; \
	done
endef

# ─── Help ────────────────────────────────────────────────────────
help:
	@echo "$(BLUE)eidos Build System$(NC)"
	@echo ""
	@echo "$(GREEN)Setup:$(NC)"
	@echo "  bootstrap          Install development tools"
	@echo "  install            Download and verify Go dependencies"
	@echo ""
	@echo "$(GREEN)Development:$(NC)"
	@echo "  fmt                Format Go + Markdown"
	@echo "  license            Apply license headers to all Go files"
	@echo "  lint               Full lint suite (fmt + vet + golangci-lint + markdownlint)"
	@echo "  lint-md            Lint Markdown files only"
	@echo "  vet                Run go vet across all modules"
	@echo "  tidy               Run go mod tidy across all modules"
	@echo "  generate           Run go generate + fmt (uses ./cmd/eidos)"
	@echo ""
	@echo "$(GREEN)Testing:$(NC)"
	@echo "  test               Run tests with coverage across all modules"
	@echo "  test-race          Run tests with race detector"
	@echo "  test-bench         Run all benchmarks"
	@echo "  bench-baseline     Record bench/baseline.txt for regression comparison"
	@echo "  bench-regression   Diff current numbers against bench/baseline.txt via benchstat"
	@echo "  test-fuzz          Run every Fuzz* target for FUZZ_TIME (default: 30s)"
	@echo "  test-coverage      Generate HTML coverage report"
	@echo ""
	@echo "$(GREEN)Quality gates:$(NC)"
	@echo "  check              Full pre-merge gate (tidy + lint + test + coverage)"
	@echo "  check-tidy         Fail if go mod tidy produces uncommitted changes"
	@echo "  check-coverage     Enforce coverage thresholds"
	@echo "  check-vuln         Run govulncheck across all modules"
	@echo ""
	@echo "$(GREEN)Building:$(NC)"
	@echo "  build              Compile every module's source (sanity check)"
	@echo "  clean              Remove build artifacts and caches"
	@echo ""
	@echo "$(YELLOW)Modules:$(NC) $(MODULES)"
	@echo "$(YELLOW)Flags:$(NC)   FLAGS=\"-run TestFoo\"          extra flags for test commands"
	@echo "          TEST_TIMEOUT=30m              per-package go test deadline"
	@echo "          TEST_COUNT=1                  iteration count for plain tests"
	@echo "          TEST_RACE_COUNT=3             iteration count for test-race"
	@echo "          TEST_CPU=8                    -cpu=N for parallel scheduling"
	@echo "          FUZZ_TIME=5m                  per-target budget for test-fuzz"
	@echo ""
	@echo "$(RED)Naming:$(NC)  test-* runs tests; check-* enforces a quality gate"

.DEFAULT_GOAL := help

# ─── Setup ───────────────────────────────────────────────────────

bootstrap:
	@echo "$(BLUE)Installing development tools...$(NC)"
	$(GO) install mvdan.cc/gofumpt@latest
	$(GO) install github.com/daixiang0/gci@latest
	$(GO) install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
	$(GO) install golang.org/x/vuln/cmd/govulncheck@latest
	$(GO) install github.com/palantir/go-license@latest
	$(GO) install golang.org/x/perf/cmd/benchstat@latest
	@command -v markdownlint-cli2 >/dev/null 2>&1 || { \
		command -v npm >/dev/null 2>&1 && npm install -g markdownlint-cli2 || \
		echo "$(YELLOW)Install markdownlint-cli2: brew install markdownlint-cli2 (or npm install -g markdownlint-cli2)$(NC)"; \
	}
	@echo "$(GREEN)Done. Run 'pre-commit install --hook-type pre-commit --hook-type pre-push --hook-type commit-msg'$(NC)"

install:
	@echo "$(BLUE)Installing dependencies...$(NC)"
	$(call foreach_module,$(GO) mod download && $(GO) mod verify)
	@echo "$(GREEN)Done$(NC)"

# ─── Code generation ─────────────────────────────────────────────
# `go generate ./...` runs in each module. The reference binary
# `eidos` lives in a separate repo; consumers running //go:generate
# directives that invoke `eidos` install it themselves via
# `go install go.thesmos.sh/eidos-cmd/eidos@latest` (or similar).

generate:
	@echo "$(BLUE)Running code generation across modules...$(NC)"
	$(call foreach_module,$(GO) generate ./...)
	@echo "$(BLUE)Regenerating testdata golden files...$(NC)"
	@dirs=$$(grep -rl '//go:generate' --include='*.go' \
		$$(find . -type d -path '*/testdata/*' -not -path './.git/*' -not -path './vendor/*' 2>/dev/null) 2>/dev/null \
		| xargs -n1 dirname 2>/dev/null | sort -u); \
	for dir in $$dirs; do \
		echo "$(BLUE)  $$dir$(NC)"; \
		(cd $$dir && $(GO) generate ./... 2>/dev/null) || true; \
	done
	@$(MAKE) fmt
	@echo "$(GREEN)Done$(NC)"

# ─── Formatting ──────────────────────────────────────────────────

fmt: license
	@echo "$(BLUE)Formatting Go...$(NC)"
	gofumpt -l -w -extra .
	gci write --section standard --section default --section "prefix(go.thesmos.sh/eidos)" --custom-order --skip-generated .
	@echo "$(BLUE)Formatting Markdown...$(NC)"
	markdownlint-cli2 --fix "**/*.md" "#vendor" "#dist" "#node_modules" "#docs/superpowers" 2>/dev/null || true
	@echo "$(GREEN)Done$(NC)"

license:
	@echo "$(BLUE)Applying license headers...$(NC)"
	@go-license --config=.go-license.yml $(GO_FILES)
	@echo "$(GREEN)Done$(NC)"

# ─── Linting ─────────────────────────────────────────────────────

vet:
	$(call foreach_module,$(GO) vet ./...)

lint: fmt vet lint-md
	@echo "$(BLUE)Running golangci-lint...$(NC)"
	$(call foreach_module,golangci-lint run --timeout=5m ./...)
	@echo "$(BLUE)Verifying license headers...$(NC)"
	@go-license --config=.go-license.yml --verify $(GO_FILES)
	@echo "$(GREEN)Lint passed$(NC)"

lint-md:
	@echo "$(BLUE)Linting Markdown...$(NC)"
	markdownlint-cli2 "**/*.md" "#vendor" "#dist" "#node_modules" "#docs/superpowers"

# ─── Generated testdata packages ────────────────────────────────
# `go test ./...` excludes testdata/ by Go convention. Any directory
# under any `testdata/` carrying a `*_test.go` file gets tested
# explicitly. Empty in early milestones; populated as plugins ship
# golden-file regression suites.
GEN_TESTDATA := $(shell find . -path '*/testdata/*' -name '*_test.go' \
	-not -path './.git/*' -not -path './vendor/*' \
	-exec dirname {} \; 2>/dev/null | sort -u)

# ─── Testing ─────────────────────────────────────────────────────

test: test-generated
	@echo "$(BLUE)Running tests (timeout=$(TEST_TIMEOUT))...$(NC)"
	@mkdir -p $(COVERAGE_DIR)
	$(call foreach_module,$(GO) test -coverprofile=$(CURDIR)/$(COVERAGE_DIR)/$$(basename $$PWD).out -covermode=atomic -cpu=$(TEST_CPU) -count=$(TEST_COUNT) -timeout=$(TEST_TIMEOUT) $(FLAGS) ./...)
	@echo "$(GREEN)Tests passed$(NC)"

test-generated:
	@echo "$(BLUE)Running generated testdata tests...$(NC)"
	@if [ -z "$(GEN_TESTDATA)" ]; then \
		echo "$(YELLOW)  no testdata packages with tests$(NC)"; \
	else \
		for pkg in $(GEN_TESTDATA); do \
			echo "$(BLUE)  $$pkg$(NC)"; \
			$(GO) test -cover -count=1 $$pkg || exit 1; \
		done; \
	fi
	@echo "$(GREEN)Generated tests passed$(NC)"

test-race:
	@echo "$(BLUE)Running tests with race detector (count=$(TEST_RACE_COUNT), timeout=$(TEST_TIMEOUT))...$(NC)"
	$(call foreach_module,$(GO) test -race -count=$(TEST_RACE_COUNT) -timeout=$(TEST_TIMEOUT) $(FLAGS) ./...)
	@echo "$(GREEN)No races detected$(NC)"

test-bench:
	@echo "$(BLUE)Running benchmarks (timeout=$(TEST_TIMEOUT))...$(NC)"
	$(call foreach_module,$(GO) test -bench=. -run=^$$ -benchmem -timeout=$(TEST_TIMEOUT) $(FLAGS) ./...)

# bench-baseline records a fresh baseline file at bench/baseline.txt
# that subsequent runs compare against. Run this when intentional
# performance changes land — the new baseline becomes the reference.
bench-baseline:
	@mkdir -p bench
	@echo "$(BLUE)Recording bench baseline to bench/baseline.txt...$(NC)"
	@$(GO) test -bench=. -run=^$$ -benchmem -count=$(BENCH_COUNT) -timeout=$(TEST_TIMEOUT) $(FLAGS) ./... | tee bench/baseline.txt
	@echo "$(GREEN)Baseline recorded$(NC)"

# bench-regression compares current numbers to bench/baseline.txt
# via benchstat. Wire this into CI on the release branch to catch
# performance drift.
bench-regression:
	@if [ ! -f bench/baseline.txt ]; then \
		echo "$(YELLOW)bench/baseline.txt missing — run 'make bench-baseline' first$(NC)"; \
		exit 1; \
	fi
	@echo "$(BLUE)Running current benchmarks (count=$(BENCH_COUNT))...$(NC)"
	@$(GO) test -bench=. -run=^$$ -benchmem -count=$(BENCH_COUNT) -timeout=$(TEST_TIMEOUT) $(FLAGS) ./... > /tmp/bench-current.txt
	@echo "$(BLUE)Comparing against baseline...$(NC)"
	@benchstat bench/baseline.txt /tmp/bench-current.txt

# go test only fuzzes one target per invocation, so we discover every
# Fuzz* function in the tree and run them sequentially. Each gets
# FUZZ_TIME of wall-clock budget; bump it for deeper local runs.
test-fuzz:
	@echo "$(BLUE)Running fuzz tests (fuzztime=$(FUZZ_TIME))...$(NC)"
	@found=0; \
	for f in $$(grep -rl '^func Fuzz' --include='*_test.go' . 2>/dev/null); do \
		pkg=$$(dirname $$f); \
		names=$$(awk '/^func Fuzz/ { sub(/\(.*/, "", $$2); print $$2 }' $$f); \
		for fn in $$names; do \
			echo "$(BLUE)  $$pkg :: $$fn$(NC)"; \
			regex="^$$fn$$"; \
			$(GO) test -run='^$$$$' -fuzz="$$regex" -fuzztime=$(FUZZ_TIME) $$pkg || exit 1; \
			found=$$((found + 1)); \
		done; \
	done; \
	if [ $$found -eq 0 ]; then \
		echo "$(YELLOW)  no fuzz tests found$(NC)"; \
	else \
		echo "$(GREEN)$$found fuzz target(s) ran for $(FUZZ_TIME) each$(NC)"; \
	fi

test-coverage: test
	@echo "$(BLUE)Generating coverage reports...$(NC)"
	@for mod in $(MODULES); do \
		name=$$(basename $$mod); \
		if [ "$$mod" = "." ]; then name="eidos"; fi; \
		if [ -f $(COVERAGE_DIR)/$$name.out ]; then \
			$(GO) tool cover -html=$(COVERAGE_DIR)/$$name.out -o $(COVERAGE_DIR)/$$name.html; \
			echo "$(GREEN)Report: $(COVERAGE_DIR)/$$name.html$(NC)"; \
		fi \
	done

# ─── Quality gates ───────────────────────────────────────────────

check-coverage:
	@printf "$(BLUE)Running tests …$(NC) "
	@logf=$$(mktemp); \
	if $(MAKE) --no-print-directory -s test >$$logf 2>&1; then \
		printf "$(GREEN)ok$(NC)\n\n"; \
		rm -f $$logf; \
	else \
		printf "$(RED)FAILED$(NC)\n\n"; \
		cat $$logf; \
		rm -f $$logf; \
		exit 1; \
	fi

check-vuln:
	$(call foreach_module,govulncheck ./...)

# ─── Building ────────────────────────────────────────────────────
# Eidos is a library family — no binary lives in this repo. The
# reference binary moved to its own repository (consumers embed
# `cli.Run` directly when writing their own CLI). `build` is kept
# as a sanity-check target that compiles every module's source
# without producing artifacts.

build:
	@echo "$(BLUE)Compiling every module (no artifacts produced)...$(NC)"
	$(call foreach_module,$(GO) build ./...)
	@echo "$(GREEN)Done$(NC)"

# ─── Cleanup ─────────────────────────────────────────────────────
# `.eidos/` covers both the build cache (spec §14: ./.eidos/cache)
# and coverage output written by `make test`.

clean:
	@echo "$(BLUE)Cleaning...$(NC)"
	rm -rf $(BIN_DIR) .eidos/ dist/
	$(call foreach_module,$(GO) clean -cache -testcache)
	@echo "$(GREEN)Clean$(NC)"

# ─── Module hygiene ─────────────────────────────────────────────

tidy:
	$(call foreach_module,$(GO) mod tidy)

check-tidy: tidy
	@dirty=0; \
	for mod in $(MODULES); do \
		if ! git diff --quiet -- $$mod/go.mod $$mod/go.sum 2>/dev/null; then \
			echo "$(RED)$$mod: go mod tidy produced changes. Run 'make tidy' and commit.$(NC)"; \
			git diff --stat -- $$mod/go.mod $$mod/go.sum; \
			dirty=1; \
		fi; \
	done; \
	test "$$dirty" -eq 0

# ─── Release ─────────────────────────────────────────────────────
# Multi-module release coordination lives in a separate tool
# (moved out of this repo). It reads go.work, infers per-module
# bumps from conventional commits since each module's last tag,
# and creates the annotated tags. This target is intentionally
# stubbed so accidental invocations surface the new flow.

release:
	@echo "$(YELLOW)Release tagging moved to the external release tool.$(NC)"
	@echo "$(YELLOW)Run that tool from the repo root; it discovers modules via go.work.$(NC)"
	@exit 1

# ─── CI gate ─────────────────────────────────────────────────────

check: check-tidy lint test check-coverage
	@echo "$(GREEN)All checks passed$(NC)"
