PROJECT_PATH := $(CURDIR)
TOOLS_BIN_DIR := $(PROJECT_PATH)/bin
TOOLS_MOD := $(PROJECT_PATH)/tools/go.mod

BUF := $(TOOLS_BIN_DIR)/buf
PROTOC_GEN_GO := $(TOOLS_BIN_DIR)/protoc-gen-go
STRINGER := $(TOOLS_BIN_DIR)/stringer
GOLANGCI_LINT := $(TOOLS_BIN_DIR)/golangci-lint
GOFUMPT := $(TOOLS_BIN_DIR)/gofumpt
GOIMPORTS := $(TOOLS_BIN_DIR)/goimports
CARTO := $(TOOLS_BIN_DIR)/carto
MOCKGEN := $(TOOLS_BIN_DIR)/mockgen

$(TOOLS_BIN_DIR):
	@ mkdir -p $(TOOLS_BIN_DIR)

$(BUF): $(TOOLS_BIN_DIR)
	@ GOBIN=$(TOOLS_BIN_DIR) go install -modfile=$(TOOLS_MOD) github.com/bufbuild/buf/cmd/buf

$(PROTOC_GEN_GO): $(TOOLS_BIN_DIR)
	@ GOBIN=$(TOOLS_BIN_DIR) go install -modfile=$(TOOLS_MOD) google.golang.org/protobuf/cmd/protoc-gen-go

$(STRINGER): $(TOOLS_BIN_DIR)
	@ GOBIN=$(TOOLS_BIN_DIR) go install -modfile=$(TOOLS_MOD) golang.org/x/tools/cmd/stringer

$(GOLANGCI_LINT): $(TOOLS_BIN_DIR)
	@ GOBIN=$(TOOLS_BIN_DIR) go install -modfile=$(TOOLS_MOD) github.com/golangci/golangci-lint/cmd/golangci-lint

$(GOFUMPT): $(TOOLS_BIN_DIR)
	@ GOBIN=$(TOOLS_BIN_DIR) go install -modfile=$(TOOLS_MOD) mvdan.cc/gofumpt

$(GOIMPORTS): $(TOOLS_BIN_DIR)
	@ GOBIN=$(TOOLS_BIN_DIR) go install -modfile=$(TOOLS_MOD) golang.org/x/tools/cmd/goimports

$(CARTO): $(TOOLS_BIN_DIR)
	@ GOBIN=$(TOOLS_BIN_DIR) go install -modfile=$(TOOLS_MOD) github.com/schigh/carto

$(MOCKGEN): $(TOOLS_BIN_DIR)
	@ GOBIN=$(TOOLS_BIN_DIR) go install -modfile=$(TOOLS_MOD) go.uber.org/mock/mockgen

.PHONY: get-tools
get-tools: $(GQLGEN) $(GOLANGCI_LINT) $(GOFUMPT) $(GOIMPORTS) $(CARTO) $(PROTOC_GEN_GO) $(BUF) $(MOCKGEN) ## get all the tools
