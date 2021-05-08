.DEFAULT_GOAL:=help
SHELL:=/usr/bin/env bash

COLOR:=\\033[36m
NOCOLOR:=\\033[0m

.PHONY: test

test:
	go test -v ./...

build:
	go test -c -v ./...


##@ Verify
.PHONY: verify verify-golangci-lint verify-go-mod

verify: verify-golangci-lint verify-go-mod ## Runs verification scripts to ensure correct execution

verify-go-mod: ## Runs the go module linter
	./hack/verify-go-mod.sh

verify-golangci-lint: ## Runs all golang linters
	./hack/verify-golangci-lint.sh


##@ Dependencies

.SILENT: update-deps update-deps-go
.PHONY:  update-deps update-deps-go

update-deps: update-deps-go ## Update all dependencies for this repo
	echo -e "${COLOR}Commit/PR the following changes:${NOCOLOR}"
	git status --short

update-deps-go: GO111MODULE=on
update-deps-go: ## Update all golang dependencies for this repo
	go get -u -t ./...
	go mod tidy
	go mod verify
	$(MAKE) test-go-unit
	./hack/update-all.sh

##@ Helpers

.PHONY: help

help:  ## Display this help
	@awk \
		-v "col=${COLOR}" -v "nocol=${NOCOLOR}" \
		' \
			BEGIN { \
				FS = ":.*##" ; \
				printf "\nUsage:\n  make %s<target>%s\n", col, nocol \
			} \
			/^[a-zA-Z_-]+:.*?##/ { \
				printf "  %s%-15s%s %s\n", col, $$1, nocol, $$2 \
			} \
			/^##@/ { \
				printf "\n%s%s%s\n", col, substr($$0, 5), nocol \
			} \
		' $(MAKEFILE_LIST)
