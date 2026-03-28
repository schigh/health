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

# E2E test targets (requires Docker + Kind)
.PHONY: e2e
e2e: e2e-cluster e2e-build e2e-deploy e2e-test e2e-teardown ## run full E2E test suite

.PHONY: e2e-cluster
e2e-cluster: ## create Kind cluster
	@kind create cluster --name health-e2e --config e2e/kind-config.yaml --wait 60s 2>/dev/null || true

.PHONY: e2e-build
e2e-build: ## build and load E2E service images
	@for svc in gateway orders payments; do \
		docker build --build-arg SERVICE=$$svc -t health-$$svc:e2e -f e2e/Dockerfile . && \
		kind load docker-image health-$$svc:e2e --name health-e2e; \
	done

.PHONY: e2e-deploy
e2e-deploy: ## deploy services to Kind cluster
	@kubectl apply -f e2e/k8s/
	@kubectl wait --for=condition=Ready pod -l app=postgres --timeout=120s
	@kubectl wait --for=condition=Ready pod -l app=redis --timeout=120s
	@kubectl wait --for=condition=Ready pod -l app=payments --timeout=120s
	@kubectl wait --for=condition=Ready pod -l app=orders --timeout=120s
	@kubectl wait --for=condition=Ready pod -l app=gateway --timeout=120s

.PHONY: e2e-test
e2e-test: ## run E2E tests against deployed cluster
	@go test -tags e2e -v -count=1 -timeout=300s ./e2e/

.PHONY: e2e-teardown
e2e-teardown: ## tear down Kind cluster
	@kind delete cluster --name health-e2e
