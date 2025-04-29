.DEFAULT_GOAL := help
MAKEFLAGS += --silent --no-print-directory

BIN_DIR := ./bin
SCRIPTS_DIR := ./scripts
APP_NAME := nobl9-language-server
VERSION_PKG := "$(shell go list -m)/internal/version"

VERSION ?= 1.0.0-test
BRANCH ?= $(shell git rev-parse --abbrev-ref HEAD)
REVISION ?= $(shell git rev-parse --short=8 HEAD)

LDFLAGS := -s -w \
	-X $(VERSION_PKG).BuildVersion=$(VERSION) \
	-X $(VERSION_PKG).BuildGitBranch=$(BRANCH) \
	-X $(VERSION_PKG).BuildGitRevision=$(REVISION)

# renovate datasource=github-releases depName=securego/gosec
GOSEC_VERSION := v2.22.3
# renovate datasource=github-releases depName=golangci/golangci-lint
GOLANGCI_LINT_VERSION := v1.64.8
# renovate datasource=go depName=golang.org/x/vuln/cmd/govulncheck
GOVULNCHECK_VERSION := v1.1.4
# renovate datasource=go depName=golang.org/x/tools/cmd/goimports
GOIMPORTS_VERSION := v0.32.0

# Check if the program is present in $PATH and install otherwise.
# ${1} - oneOf{binary,yarn}
# ${2} - program name
define _ensure_installed
	LOCAL_BIN_DIR=$(BIN_DIR) $(SCRIPTS_DIR)/ensure_installed.sh "${1}" "${2}"
endef

# Install Go binary using 'go install' with an output directory set via $GOBIN.
# ${1} - repository url
define _install_go_binary
	GOBIN=$(realpath $(BIN_DIR)) go install "${1}"
endef

# Print Makefile target step description for check.
# Only print 'check' steps this way, and not dependent steps, like 'install'.
# ${1} - step description
define _print_step
	printf -- '------\n%s...\n' "${1}"
endef

.PHONY: build
## Build nobl9-language-server binary.
build:
	$(call _print_step,Building server binary)
	go build -ldflags="$(LDFLAGS)" -o $(BIN_DIR)/$(APP_NAME) ./cmd/nobl9-language-server/

.PHONY: test test/go test/neovim
## Run all unit tests.
test: test/go test/neovim

## Run Go unit tests.
test/go:
	$(call _print_step,Running Go tests)
	go test -race -cover ./...

## Run plenary unit tests in headless Neovim instance.
test/neovim:
	$(call _print_step,Running plenary Neovim \(Lua\) tests)
	nvim \
		--headless \
		--noplugin \
		-i NONE \
		-u tests/lua/bootstrap.lua \
		-c "PlenaryBustedDirectory tests/lua/specs { minimal_init = 'tests/lua/minimal_init.lua', timeout = 50000 }"

.PHONY: nvim-open
## Open Neovim with the LSP server.
nvim-open: install/binary
	$(call _print_step,Opening Neovim with minimal config)
	nvim --clean -u ./neovim-config/init.lua service.yaml

.PHONY: check check/vet check/lint check/gosec check/spell check/trailing check/markdown check/format check/generate check/vulns
## Run all checks.
check: check/vet check/lint check/gosec check/spell check/trailing check/markdown check/format check/generate check/vulns

check/vet:
	$(call _print_step,Running go vet)
	go vet ./...

check/lint:
	$(call _print_step,Running golangci-lint)
	$(call _ensure_installed,binary,golangci-lint)
	$(BIN_DIR)/golangci-lint run

check/gosec:
	$(call _print_step,Running gosec)
	$(call _ensure_installed,binary,gosec)
	$(BIN_DIR)/gosec -exclude-generated -quiet ./...

check/spell:
	$(call _print_step,Verifying spelling)
	$(call _ensure_installed,yarn,cspell)
	yarn --silent cspell --no-progress '**/**'

check/trailing:
	$(call _print_step,Looking for trailing whitespaces)
	$(SCRIPTS_DIR)/check-trailing-whitespaces.bash

check/markdown:
	$(call _print_step,Verifying Markdown files)
	$(call _ensure_installed,yarn,markdownlint)
	yarn --silent markdownlint '**/*.md' \
		--ignore '**/testdata/**' \
		--ignore 'internal/hover/templates/*' \
		--ignore node_modules

check/vulns:
	$(call _print_step,Running govulncheck)
	$(call _ensure_installed,binary,govulncheck)
	$(BIN_DIR)/govulncheck ./...

check/generate:
	$(call _print_step,Checking if generated code matches the provided definitions)
	$(SCRIPTS_DIR)/check-generate.sh

check/format:
	$(call _print_step,Checking if files are formatted)
	$(SCRIPTS_DIR)/check-formatting.sh

generate/code:
	$(call _print_step,Generating Go code)
	go generate ./...

format/go:
	$(call _print_step,Formatting Go files)
	$(call _ensure_installed,binary,goimports)
	gofmt -l -w -s .
	$(BIN_DIR)/goimports -local=github.com/nobl9/nobl9-language-server -w .

format/cspell:
	$(call _print_step,Formatting cspell.yaml configuration (words list))
	$(call _ensure_installed,yarn,yaml)
	yarn --silent format-cspell-config

install/binary:
	$(call _print_step,Installing server binary)
	go install -gcflags="all=-N -l" -ldflags="$(LDFLAGS)" ./cmd/nobl9-language-server/

install/yarn:
	$(call _print_step,Installing yarn dependencies)
	yarn --silent install

install/golangci-lint:
	$(call _print_step,Installing golangci-lint)
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh |\
		sh -s -- -b $(BIN_DIR) $(GOLANGCI_LINT_VERSION)

install/gosec:
	$(call _print_step,Installing gosec)
	curl -sfL https://raw.githubusercontent.com/securego/gosec/master/install.sh |\
		sh -s -- -b $(BIN_DIR) $(GOSEC_VERSION)

install/govulncheck:
	$(call _print_step,Installing govulncheck)
	$(call _install_go_binary,golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION))

install/goimports:
	$(call _print_step,Installing goimports)
	$(call _install_go_binary,golang.org/x/tools/cmd/goimports@$(GOIMPORTS_VERSION))

.PHONY: help
## Print this help message.
help:
	$(SCRIPTS_DIR)/makefile-help.awk $(MAKEFILE_LIST)
