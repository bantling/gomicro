# SPDX-License-Identifier: Apache-2.0

reviveConfigFile = revive.toml
reviveTestConfigFile = revive_test.toml

.PHONY: all
all: check-go check-path install-tools compile lint format test

.PHONY: check-go
check-go:
	which go > /dev/null || { \
		echo "go is not installed"; \
		exit 1; \
	}

.PHONY: check-path
check-path:
	echo $$PATH | grep -q "`go env GOPATH`/bin" || { \
		echo "`go env GOPATH`/bin is not in PATH"; \
		exit 1; \
	}

.PHONY: install-tools
install-tools:
	which revive 2> /dev/null || go get -u github.com/mgechev/revive

.PHONY: compile
compile:
	[ -n "$(mod)" ] && go build "-mod=$(mod)" ./... || go build ./...

.PHONY: lint
lint:
	go vet ./... || exit $$?; \
	reviveConfigOpt=""; \
	[ -f "$(reviveConfigFile)" ] && reviveConfigOpt="-config `readlink -f $(reviveConfigFile)`"; \
	reviveOut=""; \
	for pkg in `go list -f '{{.Dir}}' ./...`; do \
	  reviveOut="$$reviveOut`cd $$pkg; revive $$reviveConfigOpt $$(go list -f '{{.GoFiles}}' | tr -d '[]')`"; \
	done; \
	echo "$$reviveOut"; \
	case "$$reviveOut" in \
	  ""|*"\n"*) ;; \
	  *) exit 1;; \
	esac; \
	reviveTestConfigOpt=""; \
	[ -f "$(reviveTestConfigFile)" ] && reviveTestConfigOpt="-config `readlink -f $(reviveTestConfigFile)`"; \
	reviveOut=""; \
	for pkg in `go list -f '{{.Dir}}' ./...`; do \
	  reviveOut="$$reviveOut`cd $$pkg; revive $$reviveTestConfigOpt $$(go list -f '{{.TestGoFiles}}' | tr -d '[]')`"; \
	done; \
	echo "$$reviveOut"; \
	case "$$reviveOut" in \
	  ""|*"\n"*) ;; \
	  *) exit 1;; \
	esac

.PHONY: format
format:
	for pkg in `go list -f '{{.Dir}}' ./...`; do \
	  cd $$pkg; \
	  gofmt -s -w $$(go list -f '{{.GoFiles}} {{.TestGoFiles}}' | tr -d '[]'); \
	done

.PHONY: test
test:
	modOpt=""; \
	[ -n "$(mod)" ] && modOpt="-mod=$(mod)"; \
	testOpt="-count=$${count:-1}"; \
	[ -n "$(run)" ] && testOpt="$$testOpt -run $(run)"; \
	go test $$modOpt -v $$testOpt ./...
