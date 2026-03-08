# SmartBed Backend — Makefile
# ────────────────────────────────────────────────────────

APP_NAME     := smartbed
MAIN_PKG     := ./cmd/server
BUILD_DIR    := ./bin
MIGRATIONS   := ./migrations
SEEDS        := ./seeds/baseline_ranges.sql
DB_DSN       ?= postgres://smartbed:smartbed_secret@localhost:5432/smartbed?sslmode=disable

.PHONY: all build run test test-unit test-integration clean \
        migrate-up migrate-down seed keys swagger docker-up docker-down

# ── Build ─────────────────────────────────────────────────────────────────────

all: build

build:
	@echo "Building $(APP_NAME)..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 go build -ldflags="-w -s" -o $(BUILD_DIR)/$(APP_NAME) $(MAIN_PKG)
	@echo "Binary: $(BUILD_DIR)/$(APP_NAME)"

run:
	@go run $(MAIN_PKG)

# ── Tests ─────────────────────────────────────────────────────────────────────

test: test-unit

test-unit:
	@echo "Running unit tests..."
	go test ./tests/unit/... -v -cover -race

test-integration:
	@echo "Running integration tests (requires Docker)..."
	go test ./tests/integration/... -v -tags integration -timeout 120s

# ── Database ──────────────────────────────────────────────────────────────────

migrate-up:
	@echo "Applying migrations..."
	migrate -path $(MIGRATIONS) -database "$(DB_DSN)" up

migrate-down:
	@echo "Rolling back last migration..."
	migrate -path $(MIGRATIONS) -database "$(DB_DSN)" down 1

migrate-status:
	migrate -path $(MIGRATIONS) -database "$(DB_DSN)" version

seed:
	@echo "Seeding baseline ranges..."
	psql "$(DB_DSN)" -f $(SEEDS)

# ── JWT Keys ──────────────────────────────────────────────────────────────────

keys:
	@mkdir -p keys
	@echo "Generating RSA-2048 JWT keys..."
	openssl genrsa -out keys/private.pem 2048
	openssl rsa -in keys/private.pem -pubout -out keys/public.pem
	@echo "Keys written to ./keys/"

# ── Docs ──────────────────────────────────────────────────────────────────────

swagger:
	@echo "Generating Swagger docs..."
	swag init -g $(MAIN_PKG)/main.go -o ./docs

# ── Docker ────────────────────────────────────────────────────────────────────

docker-up:
	docker compose up -d --build

docker-down:
	docker compose down

docker-logs:
	docker compose logs -f api

# ── Utilities ─────────────────────────────────────────────────────────────────

clean:
	@rm -rf $(BUILD_DIR)
	@echo "Cleaned build artifacts"

lint:
	golangci-lint run ./...

fmt:
	gofmt -w .
	goimports -w .

tidy:
	go mod tidy
