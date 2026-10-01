SQLC_VERSION := v1.31.1

.PHONY: clean generate gen-sqlc check-sqlc lint test test-integration verify verify-down

generate: clean gen-sqlc
	@go generate ./...

gen-sqlc: check-sqlc
	cd examples && sqlc generate

check-sqlc:
	@if ! command -v sqlc > /dev/null 2>&1; then \
		echo "sqlc could not be found"; \
		echo "Installing sqlc $(SQLC_VERSION)"; \
		go install github.com/sqlc-dev/sqlc/cmd/sqlc@$(SQLC_VERSION); \
	elif ! sqlc version 2> /dev/null | grep -q $(SQLC_VERSION); then \
		echo "Incorrect version of sqlc found"; \
		echo "Installing sqlc $(SQLC_VERSION)"; \
		go install github.com/sqlc-dev/sqlc/cmd/sqlc@$(SQLC_VERSION); \
	else \
		echo "Required sqlc version $(SQLC_VERSION) is already installed"; \
	fi

lint:
	golangci-lint run ./...

test:
	go test -race ./...

test-integration:
	cd examples && go test -race -tags integration -timeout 300s ./...

verify:
	cd examples && docker compose up --build -d
	@echo "Grafana: http://localhost:$${GRAFANA_PORT:-3000}/d/sqlc-pgx-monitoring"

verify-down:
	cd examples && docker compose down

clean:
	find . -type f -name "*.go" -exec grep -qE "// Code generated .* DO NOT EDIT\." {} \; -delete
