.PHONY: all build run test clean migrate

APP_NAME=antrader
BUILD_DIR=bin
CMD_DIR=cmd/server

all: build

build:
	@echo "Building $(APP_NAME)..."
	@cd backend && go build -o ../$(BUILD_DIR)/$(APP_NAME) ./$(CMD_DIR)

run:
	@echo "Running $(APP_NAME)..."
	@cd backend && go run ./$(CMD_DIR)

test:
	@echo "Running tests..."
	@cd backend && go test -v ./...

coverage:
	@echo "Running tests with coverage..."
	@cd backend && go test -count=1 -race -short -coverprofile=coverage.out -covermode=atomic ./...
	@cd backend && go tool cover -func=coverage.out | tail -1

coverage-html:
	@echo "Generating coverage HTML report..."
	@cd backend && go test -count=1 -race -short -coverprofile=coverage.out -covermode=atomic ./...
	@cd backend && go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: backend/coverage.html"

security-scan:
	@echo "=== gosec ==="
	@cd backend && gosec -quiet -severity medium ./... || echo "gosec: WARNING (install: go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest)"
	@echo "=== trivy fs ==="
	@trivy fs --severity HIGH,CRITICAL --scanners vuln ./backend 2>&1 || echo "trivy: WARNING (install: https://aquasecurity.github.io/trivy/)"

clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)

migrate:
	@echo "Running migrations..."
	@set -a; . ./.env; set +a; \
	  PGPASSWORD=$$DB_PASSWORD psql -U $${DB_USER:-ant} -d $${DB_NAME:-ant} -h localhost -f backend/migrations/001_init.up.sql

deps:
	@echo "Installing dependencies..."
	@cd backend && go mod download
	@cd backend && go mod tidy

fmt:
	@echo "Formatting code..."
	@cd backend && go fmt ./...

lint:
	@echo "Linting code..."
	@cd backend && go vet ./...

docker-up:
	@echo "Starting Docker containers..."
	@docker compose up -d

docker-down:
	@echo "Stopping Docker containers..."
	@docker compose down

docker-build:
	@echo "Building Docker images..."
	@docker compose build

env-check:
	@echo "Checking .env against .env.example..."; \
	test -f .env || { echo "ERROR: .env not found. Run: cp .env.example .env"; exit 1; }; \
	MISSING=0; \
	for key in $$(grep -E '^[A-Z][A-Z0-9_]+=' .env.example | sed 's/=.*//' | sort -u); do \
		if ! grep -qE "^$${key}=" .env 2>/dev/null; then \
			echo "MISSING: $${key}"; \
			MISSING=1; \
		fi; \
	done; \
	if [ $$MISSING -ne 0 ]; then \
		echo "ERROR: missing required env keys. See .env.example for expected variables."; \
		exit 1; \
	fi; \
	echo "All env keys present ✅"

.PHONY: proto-tools proto check-lines verify

proto-tools:
	@echo "Installing proto generation toolchain..."
	@cd tools/proto-gen && npm ci
	@cd frontend && npm ci
	@mkdir -p tools/proto-gen/bin
	@# protoc-gen-connect-go@v1.19.1 requires Go >= 1.24 (see connect-go go.mod).
	@cd backend && GOBIN="$(CURDIR)/tools/proto-gen/bin" go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.11
	@cd backend && GOBIN="$(CURDIR)/tools/proto-gen/bin" go install connectrpc.com/connect/cmd/protoc-gen-connect-go@v1.19.1

proto:
	@echo "Generating protobuf code (Go + TS)..."
	@PATH="$(CURDIR)/tools/proto-gen/bin:$(CURDIR)/frontend/node_modules/.bin:$(CURDIR)/tools/proto-gen/node_modules/.bin:$$PATH" buf generate

check-lines:
	@echo "Checking file line limits..."
	@python3 scripts/check-file-lines.py

verify:
	@echo "Verifying repo (proto + line limits + go test)..."
	@$(MAKE) proto
	@$(MAKE) check-lines
	@cd backend && go test ./...

# ── RTK (Rust Token Killer) ──────────────────────────────────────────
# Token-optimized CLI proxy. All shell output filtered for LLM context.
# Requires: rtk >= 0.40.0 (installed at /root/.local/bin/rtk)
RTK := rtk

rtk-help:
	@$(RTK) --help

rtk-gain:
	@$(RTK) gain

rtk-gain-history:
	@$(RTK) gain --history

rtk-test:
	@cd backend && $(RTK) test go test ./...

rtk-build:
	@cd backend && $(RTK) err go build -o ../bin/antrader ./cmd/server

rtk-lint:
	@cd backend && $(RTK) err go vet ./...

rtk-fmt:
	@cd backend && $(RTK) err go fmt ./...

rtk-git-status:
	@$(RTK) git status

rtk-git-diff:
	@$(RTK) diff

rtk-git-log:
	@$(RTK) git log

rtk-ls:
	@$(RTK) ls

rtk-tree:
	@$(RTK) tree

rtk-deps:
	@$(RTK) deps

rtk-env:
	@$(RTK) env

rtk-grep:
	@$(RTK) grep

# ── v2 ClickHouse ─────────────────────────────────────────────────────
.PHONY: migrate-ch

sqlc:
	@echo "Generating sqlc code..."
	@cd backend && sqlc generate 2>/dev/null || echo "sqlc: generate skipped (tool not installed)"

migrate-ch:
	@echo "Running ClickHouse migrations..."
	@cd backend && go run ./cmd/migrate-ch/.

# ── Card verification ─────────────────────────────────────────────────
.PHONY: verify-card

verify-card:
	@test -n "$$CARD_ID" || { echo "ERROR: CARD_ID not set. Usage: CARD_ID=M7.X-Y make verify-card"; exit 1; }
	@echo "=== Verifying card $$CARD_ID ==="
	@test -f "docs/handover/verify-$$CARD_ID.log" || { echo "FAIL: handover log missing"; exit 1; }
	@test $$(wc -l < "docs/handover/verify-$$CARD_ID.log") -ge 20 || { echo "FAIL: log < 20 lines"; exit 1; }
	@cd backend && go build ./... || { echo "FAIL: backend build"; exit 1; }
	@cd backend && go test -race ./internal/... || { echo "FAIL: tests"; exit 1; }
	@echo "=== Card $$CARD_ID verified OK ==="

# ── Strict card verification (AGENT.md §0.3) ─────────────────────────
.PHONY: verify-cards-strict detect-stubs detect-skip-tests detect-orphan-test-claims

# Usage: make verify-cards-strict MILESTONE=M10 [APPLY=1]
verify-cards-strict:
	@test -n "$(MILESTONE)" || { echo "ERROR: MILESTONE not set. Usage: make verify-cards-strict MILESTONE=M10"; exit 1; }
	@if [ "$(APPLY)" = "1" ]; then \
		bash scripts/verify-cards-strict.sh $(MILESTONE) --apply; \
	else \
		bash scripts/verify-cards-strict.sh $(MILESTONE); \
	fi

# 反 stub 探测：扫描生产代码（非 _test.go）中的偷工关键词
detect-stubs:
	@echo "=== detect-stubs: scanning backend/{cmd,internal} for stub keywords ==="
	@HITS=$$(grep -rEn '(printf|Printf|Println|Print|Errorf|errors\.New|fmt\.Errorf)\(\s*"[^"]*\b(stub|TODO|placeholder|not (wired|(yet )?implemented))\b' \
		backend/cmd backend/internal --include='*.go' --exclude='*_test.go' 2>/dev/null | wc -l); \
	echo "Hits: $$HITS"; \
	if [ "$$HITS" -gt 0 ]; then \
		grep -rEn '(printf|Printf|Println|Print|Errorf|errors\.New|fmt\.Errorf)\(\s*"[^"]*\b(stub|TODO|placeholder|not (wired|(yet )?implemented))\b' \
			backend/cmd backend/internal --include='*.go' --exclude='*_test.go' 2>/dev/null; \
		echo "FAIL: $$HITS stub hits found"; exit 1; \
	fi; \
	echo "OK: 0 hits"

# 列出无 milestone 引用的 t.Skip
detect-skip-tests:
	@echo "=== detect-skip-tests: t.Skip without milestone reference ==="
	@grep -rEn 't\.Skip(f)?\(' backend --include='*_test.go' 2>/dev/null \
		| grep -vE '将在卡片\s*M[0-9]+\.[0-9Z]+-[0-9]+\s*中实施' \
		| tee /tmp/orphan-skips.txt; \
	N=$$(wc -l < /tmp/orphan-skips.txt 2>/dev/null || echo 0); \
	if [ "$$N" -gt 0 ]; then echo "FAIL: $$N orphan skips"; exit 1; fi; \
	echo "OK: 0 orphan skips"

# 卡片声明的 Test 函数但代码中不存在
detect-orphan-test-claims:
	@echo "=== detect-orphan-test-claims: cards claim Test funcs that don't exist ==="
	@grep -oE 'Test[A-Z][A-Za-z0-9_]+' docs/plan/ROADMAP.md | sort -u > /tmp/claimed.txt
	@grep -rhoE '^func\s+Test[A-Z][A-Za-z0-9_]+' backend --include='*_test.go' \
		| awk '{print $$2}' | sort -u > /tmp/existing.txt
	@MISSING=$$(comm -23 /tmp/claimed.txt /tmp/existing.txt); \
	if [ -n "$$MISSING" ]; then \
		echo "FAIL: claimed in ROADMAP but missing in code:"; echo "$$MISSING"; exit 1; \
	fi; \
	echo "OK: all claimed Test funcs exist"

# ── R0 验收防伪五件套 ──────────────────────────────────────────────────
.PHONY: detect-deadcode detect-fakecomplete detect-layering detect-spec-drift detect-all

# R0-2: 扫 internal/ 下 0 import 的死包
detect-deadcode:
	@bash scripts/detect-deadcode.sh

# R0-3: 扫 ROADMAP ☑ 卡片是否有 handover log
detect-fakecomplete:
	@bash scripts/detect-fakecomplete.sh

# R0-4: 扫 connect/*.go 是否直接 import pgxpool/sqlx/clickhouse
detect-layering:
	@bash scripts/detect-layering.sh

# R0-5: 比对 spec LOC 限制 vs 实际 wc -l
detect-spec-drift:
	@bash scripts/detect-spec-drift.sh

# R0 五件套全跑
detect-all: detect-stubs detect-deadcode detect-fakecomplete detect-layering detect-spec-drift
	@echo "=== R0 detect-all: all 5 checks PASSED ==="

# ── Help ──────────────────────────────────────────────────────────────
.PHONY: help

# ── CI Nightly (M10-BASE-A3) ───────────────────────────────────────────
.PHONY: ci-nightly

ci-nightly:
	@echo "=== ci-nightly: md-doctor + slo-report ==="
	@cd backend && go build -o /tmp/md-doctor ./cmd/md-doctor/
	@cd backend && go build -o /tmp/slo-report ./cmd/slo-report/
	@echo "--- md-doctor all (24h) ---"
	@/tmp/md-doctor all --window 24h --strict --output json 2>&1 || echo "md-doctor: WARNING (see above)"
	@echo "--- slo-report (24h) ---"
	@/tmp/slo-report --window 24h --strict 2>&1 || echo "slo-report: WARNING (see above)"
	@echo "=== ci-nightly complete ==="

help:
	@echo "ant Makefile targets:"
	@echo "  ci-nightly     run md-doctor + slo-report (daily cron)"
	@echo "  coverage       run tests with coverage summary"
	@echo "  coverage-html  generate coverage HTML report"
	@echo "  security-scan  run gosec + trivy filesystem scan"
	@echo "  build          compile backend binary"
	@echo "  test           run all tests"
	@echo "  migrate        run PostgreSQL migrations"
	@echo "  migrate-ch     run ClickHouse migrations (v2)"
	@echo "  verify-card    verify a ROADMAP card (set CARD_ID=M7.X-Y)"
	@echo "  proto          generate protobuf (Go + TS)"
	@echo "  docker-up      start all containers"
	@echo "  docker-down    stop all containers"
	@echo "  lint           run linters"
	@echo "  deploy         gray-release deploy"
	@echo "  backup         database backup"
	@echo "  status         health check overview"

# ── M6 deployment & ops targets ───────────────────────────────────────

.PHONY: deploy backup bench status

# M6-3: Gray-release deploy — build all images, restart affected containers.
deploy: build
	@echo "Starting gray-release deploy..."
	docker compose up -d --remove-orphans
	@echo "Waiting for health checks..."
	@sleep 5
	@docker compose ps --format "table {{.Name}}\t{{.Status}}" | grep -E 'ant-|Name'
	@echo "Deploy complete. Run 'make status' to verify."

# M6-2: Database backup.
backup:
	@./scripts/backup-db.sh

# M6-5: Smoke benchmark.
bench:
	@./scripts/bench-health.sh

# M6-4: Health status overview.
status:
	@echo "=== ant container health ==="
	@docker compose ps --format "table {{.Name}}\t{{.Status}}\t{{.Ports}}"
	@echo ""
	@echo "=== health check endpoints ==="
	@for url in \
		http://localhost:8080/healthz \
		http://localhost:8080/health \
		http://localhost:8081/healthz; do \
		printf "  %-45s " "$$url"; \
		curl -sf -o /dev/null "$$url" && echo "✅" || echo "❌"; \
	done
