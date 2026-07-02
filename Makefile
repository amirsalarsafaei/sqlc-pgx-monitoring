SQLC_VERSION := v1.29.0

.PHONY: clean generate gen-mocks gen-sqlc check-sqlc test test-integration

generate: clean gen-sqlc
	@go generate ./...

gen-sqlc: check-sqlc
	cd examples && sqlc generate

check-sqlc:
	@if ! command -v sqlc &> /dev/null; then \
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

test:
	@go test -v ./...

test-integration:
	@echo "Running integration tests (testcontainers starts a throwaway PostgreSQL; Docker must be running)..."
	cd examples && go test -v -tags integration -timeout 300s ./...

clean:
	find . -type f -name "*.go"  -exec grep -qE "// Code generated .* DO NOT EDIT\." {} \; -delete

