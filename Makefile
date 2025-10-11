lint:
	@echo "\n🔎  Linting ..."
	golangci-lint run

vulncheck:
	@echo "\n🕵  Checking for vulnerabilities ..."
	govulncheck ./...

build: vulncheck lint
	@echo "\n⚙️  Building ..."
	go build -v ./...

build-only:
	@echo "\n⚙️  Building only ..."
	go build -v ./...

test:
	@echo "\n🚦  Testing ..."
	go test -count 1 -failfast -race -v ./...

test-docker:
	@bash -c '\
		set -e; \
		cleanup() { \
			echo "🚦🔌 ⬇ 🐳  Stopping PostgreSQL container for tests ..."; \
			docker compose -f docker-compose.test.yml down -v; \
		}; \
		trap cleanup EXIT; \
		echo "🚦🔌 ⬆ 🐳  Starting PostgreSQL container for tests ..."; \
		docker compose -f docker-compose.test.yml up -d; \
		echo "🚦🔌 ⏳ 🐳  Waiting for PostgreSQL to be ready ..."; \
		sleep 2; \
		docker compose -f docker-compose.test.yml exec -T gosmig_test_postgres sh -c "while ! pg_isready -U gosmig -d gosmig; do echo \"Waiting for PostgreSQL...\"; sleep 2; done"; \
		echo "🚦🔌 🏃 🐳  Running tests with Docker PostgreSQL ..."; \
		go test -count 1 -failfast -race -v ./...; \
	'

test-docker-down:
	@echo "\n🚦🔌 ⬇ 🐳  Stopping PostgreSQL container for tests ..."
	docker compose -f docker-compose.test.yml down -v

test-gh-workflow:
	act --job checks
