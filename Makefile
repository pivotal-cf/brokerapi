###### Help ###################################################################

.DEFAULT_GOAL = help

GO-VERSION = 1.21.4
GO-VER = go$(GO-VERSION)

GO=go
GOFMT=gofmt

.PHONY: help

help:  ## list Makefile targets
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

###### Targets ################################################################

test: deps-go-binary download fmt vet ginkgo ## Runs all build, static analysis, and test steps

download: ## Download dependencies
	${GO} mod download

vet: ## Run static code analysis
	${GO} vet ./...
	${GO} run honnef.co/go/tools/cmd/staticcheck ./...

ginkgo: ## Run tests using Ginkgo
	${GO} run github.com/onsi/ginkgo/v2/ginkgo -r

fmt: ## Checks that the code is formatted correctly
	@@if [ -n "$$(${GOFMT} -s -e -l -d .)" ]; then                   \
		echo "gofmt check failed: run 'gofmt -d -e -l -w .'"; \
		exit 1;                                               \
	fi

generate: ## Generates the fakes using counterfeiter
	${GO} generate ./...

.PHONY: deps-go-binary
deps-go-binary:
ifeq ($(SKIP_GO_VERSION_CHECK),)
	@@if [ "$$($(GO) version | awk '{print $$3}')" != "${GO-VER}" ]; then \
		echo "Go version does not match: expected: ${GO-VER}, got $$($(GO) version | awk '{print $$3}')"; \
		exit 1; \
	fi
endif
