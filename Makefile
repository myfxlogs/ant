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
	@cd backend && go run ./$(CMD_DIR) -config configs/config.yaml

test:
	@echo "Running tests..."
	@cd backend && go test -v ./...

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
	@mkdir -p tools/proto-gen/bin
	@# protoc-gen-connect-go@v1.19.1 requires Go >= 1.24 (see connect-go go.mod).
	@cd backend && GOBIN="$(CURDIR)/tools/proto-gen/bin" go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.35.2
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

# ── Help ──────────────────────────────────────────────────────────────
.PHONY: help

help:
	@echo "ant Makefile targets:"
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
