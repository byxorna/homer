PACKAGE  = github.com/byxorna/homer
BINARY_CLIENT   = homer-client
BINARY_SERVER   = homer-server
DATE    ?= $(shell date +%FT%T%z)
VERSION ?= $(shell git describe --tags --always --dirty --match=v* 2> /dev/null || \
	    cat $(CURDIR)/.version 2> /dev/null || echo v0)
#GOPATH   = $(CURDIR)/.gopath~
BIN      = $(GOPATH)/bin
BASE     = $(GOPATH)/src/$(PACKAGE)
PKGS     = $(or $(PKG),$(shell cd $(BASE) && env GOPATH=$(GOPATH) $(GO) list ./... | grep -v "^$(PACKAGE)/vendor/"))
TESTPKGS = $(shell env GOPATH=$(GOPATH) $(GO) list -f '{{ if or .TestGoFiles .XTestGoFiles }}{{ .ImportPath }}{{ end }}' $(PKGS))
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
client: fmt | $(BASE) ; $(info $(M) building client executable…) @ ## Build program binary
	$Q cd $(BASE) && $(GO) build \
	  -tags release \
	  -ldflags '-X $(PACKAGE)/internal/version.Version=$(VERSION) -X $(PACKAGE)/internal/version.BuildDate=$(DATE) -X $(PACKAGE)/internal/version.Package=$(PACKAGE)' \
	  -o bin/$(BINARY_CLIENT) ./cmd/client \
		&& chmod +x bin/$(BINARY_CLIENT) \
	  && echo "Built bin/$(BINARY_CLIENT): $(VERSION) $(DATE)"
server: fmt | $(BASE) ; $(info $(M) building server executable…) @ ## Build program binary
	$Q cd $(BASE) && $(GO) build \
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
	$Q cd $(BASE) && cp bin/$(BINARY_CLIENT) bin/$(BINARY_CLIENT)-$(VERSION)
	$Q echo Tagged $(GOOS)/$(GOARCH) bin/$(BINARY_CLIENT)-$(VERSION)
	$(info $(M) tagging $(BINARY_SERVER) as $(VERSION)…) @ ## rename build to include version
	$Q cd $(BASE) && cp bin/$(BINARY_SERVER) bin/$(BINARY_SERVER)-$(VERSION)
	$Q echo Tagged $(GOOS)/$(GOARCH) bin/$(BINARY_SERVER)-$(VERSION)

.PHONY: deploy-release
deploy-release: release
	$(info $(M) deploying $(BINARY)-$(VERSION) to $(FSHOST):$(FSPATH)…)
	$Q cd $(BASE)
	scp bin/$(BINARY)-$(VERSION) $(FSHOST):$(FSPATH)/
	$Q echo Deployed release to https://$(FSHOST)/$(FSWWWPATH)/$(BINARY)-$(VERSION)

# Tests

TEST_TARGETS := test-default test-bench test-short test-verbose test-race
.PHONY: $(TEST_TARGETS) test-xml check test tests
test-bench:   ARGS=-run=__absolutelynothing__ -bench=. ## Run benchmarks
test-short:   ARGS=-short        ## Run only short tests
test-verbose: ARGS=-v            ## Run tests in verbose mode with coverage reporting
test-race:    ARGS=-race         ## Run tests with race detector
$(TEST_TARGETS): NAME=$(MAKECMDGOALS:test-%=%)
$(TEST_TARGETS): test
check test tests: fmt lint | $(BASE) ; $(info $(M) running $(NAME:%=% )tests…) @ ## Run tests
	$Q cd $(BASE) && $(GO) test -timeout $(TIMEOUT)s $(ARGS) $(TESTPKGS)

test-xml: fmt lint | $(BASE) $(GO2XUNIT) ; $(info $(M) running $(NAME:%=% )tests…) @ ## Run tests with xUnit output
	$Q cd $(BASE) && 2>&1 $(GO) test -timeout 20s -v $(TESTPKGS) | tee test/tests.output
	$(GO2XUNIT) -fail -input test/tests.output -output test/tests.xml

COVERAGE_MODE = atomic
COVERAGE_PROFILE = $(COVERAGE_DIR)/profile.out
COVERAGE_XML = $(COVERAGE_DIR)/coverage.xml
COVERAGE_HTML = $(COVERAGE_DIR)/index.html
.PHONY: test-coverage test-coverage-tools
test-coverage-tools: | $(GOCOVMERGE) $(GOCOV) $(GOCOVXML)
test-coverage: COVERAGE_DIR := $(CURDIR)/test/coverage.$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
test-coverage: fmt lint test-coverage-tools | $(BASE) ; $(info $(M) running coverage tests…) @ ## Run coverage tests
	$Q mkdir -p $(COVERAGE_DIR)/coverage
	$Q cd $(BASE) && for pkg in $(TESTPKGS); do \
	  $(GO) test \
	    -coverpkg=$$($(GO) list -f '{{ join .Deps "\n" }}' $$pkg | \
	        grep '^$(PACKAGE)/' | grep -v '^$(PACKAGE)/vendor/' | \
	        tr '\n' ',')$$pkg \
	    -covermode=$(COVERAGE_MODE) \
	    -coverprofile="$(COVERAGE_DIR)/coverage/`echo $$pkg | tr "/" "-"`.cover" $$pkg ;\
	 done
	$Q $(GOCOVMERGE) $(COVERAGE_DIR)/coverage/*.cover > $(COVERAGE_PROFILE)
	$Q $(GO) tool cover -html=$(COVERAGE_PROFILE) -o $(COVERAGE_HTML)
	$Q $(GOCOV) convert $(COVERAGE_PROFILE) | $(GOCOVXML) > $(COVERAGE_XML)


.PHONY: fmt
fmt: ; $(info $(M) running gofmt…) @ ## Run gofmt on all source files
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
