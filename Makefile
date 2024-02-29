.DELETE_ON_ERROR:
.DEFAULT_GOAL := help
_YELLOW=\033[0;33m
_GREEN=\033[0;36m
_NC=\033[0m

include tools/tools.mk

export PATH := $(TOOLS_BIN_DIR):$(PATH)
GEN_DIR := $(CURDIR)/proto/genpb

.PHONY: help
help: ## prints this help
	@ grep -hE '^[\.a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "${_YELLOW}%-24s${_NC} %s\n", $$1, $$2}'

.PHONY: clean-tools
clean-tools:
	@ rm -rf $(TOOLS_BIN_DIR)

.PHONY: lint
lint: $(GOLANGCI_LINT) ## lint you some go code but dont fix things
	@ $(GOLANGCI_LINT) run --out-format=github-actions --config=.golangci.yaml

.PHONY: lint-fix
lint-fix: $(GOLANGCI_LINT) ## lint you some go code and fix things
	@{\
  		$(GOLANGCI_LINT) cache clean && \
  		$(GOLANGCI_LINT) run --config=.golangci.yaml --fix ; \
  	}

.PHONY: fmt
fmt: $(GOFUMPT) ## format you some go code
	@{\
  		$(GOFUMPT) -w $(CURDIR) && \
  		$(GOIMPORTS) -w -local github.com/schigh/health $(CURDIR) ; \
  	}

.PHONY: buf
buf: $(BUF) ## generate proto artifacts with buf
	@{\
  		rm -rf $(GEN_DIR) && \
  		$(BUF) lint proto && \
  		$(BUF) build proto && \
  		$(BUF) generate && \
  		cp "$(GEN_DIR)/schigh/health/v1/health.pb.go" "$(CURDIR)/pkg/v1/health.pb.go" && \
  		cp "$(GEN_DIR)/schigh/health/v1/check.pb.go" "$(CURDIR)/pkg/v1/check.pb.go" && \
  		rm -rf $(GEN_DIR) ; \
  	}

.PHONY: ready
ready: ## generate all artifacts, clean, format, and vet code...get ready for a PR
	@{\
		printf "${_GREEN}%-32s${_NC} " "generating proto" && \
  		$(MAKE) buf ; \
  		printf " ✅  \n" && \
  		printf "${_GREEN}%-32s${_NC} " "formatting" && \
  		$(MAKE) fmt && \
  		printf " ✅  \n" && \
  		printf "${_GREEN}%-32s${_NC} " "linting" && \
  		$(MAKE) lint-fix && \
  		printf " ✅  \n" ; \
  	}
