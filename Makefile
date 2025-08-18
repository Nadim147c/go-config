GO ?= go
TOOL ?= $(GO) tool -modfile ./tool.go.mod

-include Makefile.local

test-lint:
	$(GO) test -v -failfast ./...
	$(GO) mod tidy
	$(GO) mod tidy -modfile ./tool.go.mod
	$(TOOL) gofumpt -l -w .
	$(TOOL) revive -config ./revive.toml $(EXCLUDE_TEST)

gen:
	$(GO) run ./generator .
	$(TOOL) gofumpt -l -w .
