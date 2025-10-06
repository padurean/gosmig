cover_file = cover.out

lint:
	@echo "\n[lint] Linting ..."
	golangci-lint run

vulncheck:
	@echo "\n[vuln] Checking for vulnerabilities ..."
	govulncheck ./...

build: vulncheck lint
	@echo "\n[build] Building ..."
	go build -v ./...

build-only:
	@echo "\n[build] Building only ..."
	go build -v ./...

test:
	@echo "\n[test] Testing ..."
	go test -count=1 -shuffle=on -failfast -race -v ./...

test-docker:
	@bash -c '\
		set -e; \
		cleanup() { \
			echo "[docker] Stopping PostgreSQL container for tests ..."; \
			docker compose -f docker-compose.test.yml down -v; \
		}; \
		trap cleanup EXIT; \
		echo "[docker] Starting PostgreSQL container for tests ..."; \
		docker compose -f docker-compose.test.yml up -d; \
		echo "[docker] Waiting for PostgreSQL to be ready ..."; \
		sleep 2; \
		docker compose -f docker-compose.test.yml exec -T gosmig_test_postgres sh -c "while ! pg_isready -U gosmig -d gosmig; do echo \"[docker] Waiting for PostgreSQL...\"; sleep 2; done"; \
		echo "[docker] Running tests with Docker PostgreSQL ..."; \
		go test -count=1 -shuffle=on -failfast -race -v ./...; \
	'

test-docker-down:
	@echo "\n[docker] Stopping PostgreSQL container for tests ..."
	docker compose -f docker-compose.test.yml down -v

test-coverage:
	@echo "\n[coverage] Testing with coverage ..."
	go test -count=1 -shuffle=on -failfast -race -v -coverprofile=${cover_file} -covermode=atomic -coverpkg=./... ./...

test-coverage-html: test-coverage
	@echo "\n[coverage] Generating HTML coverage report ..."
	go tool cover -html=${cover_file}
# Uncomment the following line to output the HTML report to a file:
# 	go tool cover -html=${cover_file} -o=cover.html

install-go-test-coverage:
	@echo "\n[tools] Installing go-test-coverage tool ..."
	go install github.com/vladopajic/go-test-coverage/v2@latest

check-coverage: test-coverage
	@echo "\n[coverage] Checking coverage thresholds ..."
	go-test-coverage --config=./.testcoverage.yml

check-coverage-only: test-coverage
	@echo "\n[coverage] Checking coverage thresholds ..."
	go-test-coverage --config=./.testcoverage.yml

test-github-workflow:
	@echo "\n[ci] Running GitHub Actions checks locally ..."
	# Requires github.com/nektos/act
	act --job checks
