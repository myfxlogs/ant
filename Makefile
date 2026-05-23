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
	@echo "Checking .env against .env.example..."
	@test -f .env || { echo "ERROR: .env not found. Run: cp .env.example .env"; exit 1; }
	@MISSING=0; \
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
	@echo "All env keys present ✅"

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
