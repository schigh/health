.DELETE_ON_ERROR:
.DEFAULT_GOAL := help
_YELLOW=\033[0;33m
_GREEN=\033[0;36m
_NC=\033[0m

.PHONY: help
help: ## prints this help
	@grep -hE '^[\.a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "${_YELLOW}%-24s${_NC} %s\n", $$1, $$2}'

.PHONY: test
test: ## run tests with race detector
	@go test -race -count=1 ./...

.PHONY: vet
vet: ## run go vet
	@go vet ./...

.PHONY: fmt
fmt: ## format code
	@gofmt -w .

.PHONY: check
check: vet test ## run all checks
