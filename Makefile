REVISER_VERSION := v3.3.1
SQLC_VERSION := v1.20.0

revise-imports: $(GOPATH)/bin/goimports-reviser
	@goimports-reviser -company-prefixes "github.com/amirsalarsafaei/" ./...

gen-sqlc: check-sqlc
	sqlc generate -f ./internal/example/db/sqlc.yaml

$(GOPATH)/bin/goimports-reviser:
	@go install -v github.com/incu6us/goimports-reviser/v3@$(REVISER_VERSION)

check-sqlc:
	@if ! command -v mockery &> /dev/null; then \
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
