PACKAGE  = github.com/byxorna/homer
BINARY_CLIENT   = homer-client
BINARY_SERVER   = homer-server
DATE    ?= $(shell date +%FT%T%z)
VERSION ?= $(shell git describe --tags --always --dirty --match=v* 2> /dev/null || \
	    cat $(CURDIR)/.version 2> /dev/null || echo v0)
BIN      = $(GOPATH)/bin
GO      = go
GODOC   = godoc
GOFMT   = gofmt
TIMEOUT = 15
V = 0
Q = $(if $(filter 1,$V),,@)
M = $(shell printf "\033[34;1m✈\033[0m")

.PHONY: all
all: client server

.PHONY: client
client: fmt
	$(info $(M) building client executable…) @ ## Build program binary
	@$(GO) build \
	  -tags release \
	  -ldflags '-X $(PACKAGE)/internal/version.Version=$(VERSION) -X $(PACKAGE)/internal/version.BuildDate=$(DATE) -X $(PACKAGE)/internal/version.Package=$(PACKAGE)' \
	  -o bin/$(BINARY_CLIENT) ./cmd/client \
		&& chmod +x bin/$(BINARY_CLIENT) \
	  && echo "Built bin/$(BINARY_CLIENT): $(VERSION) $(DATE)"
server: fmt
	$(info $(M) building server executable…) @ ## Build program binary
	@$(GO) build \
	  -tags release \
	  -ldflags '-X $(PACKAGE)/internal/version.Version=$(VERSION) -X $(PACKAGE)/internal/version.BuildDate=$(DATE) -X $(PACKAGE)/internal/version.Package=$(PACKAGE)' \
	  -o bin/$(BINARY_SERVER) ./cmd/server \
		&& chmod +x bin/$(BINARY_SERVER) \
	  && echo "Built bin/$(BINARY_SERVER): $(VERSION) $(DATE)"
.PHONY: release
release: export GOOS=linux
release: export GOARCH=amd64
release: all
	$(info $(M) tagging $(BINARY_CLIENT) as $(VERSION)…) @ ## rename build to include version
	@cp bin/$(BINARY_CLIENT) bin/$(BINARY_CLIENT)-$(VERSION)
	$Q echo Tagged $(GOOS)/$(GOARCH) bin/$(BINARY_CLIENT)-$(VERSION)
	$(info $(M) tagging $(BINARY_SERVER) as $(VERSION)…) @ ## rename build to include version
	@cp bin/$(BINARY_SERVER) bin/$(BINARY_SERVER)-$(VERSION)
	$Q echo Tagged $(GOOS)/$(GOARCH) bin/$(BINARY_SERVER)-$(VERSION)

.PHONY: deploy-release
deploy-release: release
	$(info $(M) deploying $(BINARY)-$(VERSION) to $(FSHOST):$(FSPATH)…)
	scp bin/$(BINARY)-$(VERSION) $(FSHOST):$(FSPATH)/
	$Q echo Deployed release to https://$(FSHOST)/$(FSWWWPATH)/$(BINARY)-$(VERSION)

.PHONY: fmt
fmt:
	$(info $(M) running gofmt…) @ ## Run gofmt on all source files
	@ret=0 && for d in $$($(GO) list -f '{{.Dir}}' ./... | grep -v /vendor/); do \
	  $(GOFMT) -l -w $$d/*.go || ret=$$? ; \
	 done ; exit $$ret

# Misc

.PHONY: clean
clean: ; $(info $(M) cleaning…)  @ ## Cleanup everything
	@rm -rf $(GOPATH)
	@rm -rf bin
	@rm -rf test/tests.* test/coverage.*

.PHONY: help
help:
	@grep -E '^[ a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
	  awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'

.PHONY: version
version:
	@echo $(VERSION)
