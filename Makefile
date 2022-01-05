.DEFAULT_GOAL:=help
SHELL:=/usr/bin/env bash

COLOR:=\\033[36m
NOCOLOR:=\\033[0m

.PHONY: test build docker-build docker-push sonobuoy-run sonobuoy-retrieve

##@ Build

test: ## Runs tests locally
	go test -v ./tests

build: ## Build tests in a binary
	go test -v -c -o svc-test ./tests

docker-build: ## Build project into docker container image
	docker build . -t yzaccc/k8s-service-validator:dev

docker-push: ## Push the project docker image to dockerhub
	docker push yzaccc/k8s-service-validator:dev

sonobuoy-run: ## Run k8s-service-validator as sonobuoy plugin in a cluster
	sonobuoy run --plugin sonobuoy-plugin.yaml --wait

sonobuoy-retrieve: ## Retrieve results from sonobuoy plugin
	rm -rf results
	$(eval OUTFILE=$(shell sonobuoy retrieve))
	echo $(OUTFILE)
	mkdir results && tar -xf $(OUTFILE) -C results
	cat results/plugins/k8s-service-validator-sonobuoy-plugin/*.yaml

clean: ## Clean up sonobuoy results and output binary
	rm -rf results *_sonobuoy_* svc-test

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