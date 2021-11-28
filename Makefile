.PHONY: help
help: ## Shows the available commands
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.PHONY: all
all: build run-sample ## Make everything

.PHONY: build
build: ## Build
	@go build .

.PHONY: run-sample
run-sample: build ## Run sample
	@./k8s-yaml-diff --mode full --source ./test/source.yaml --target ./test/target.yaml
