default: help

.PHONY: help
help: ## list makefile targets
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

PHONY: test
test: ## run rust tests
	cargo test

PHONY: fmt
fmt: ## format rust files
	cargo fmt

PHONY: lint
lint: ## lint rust files
	cargo clippy
