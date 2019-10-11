SHELL := /bin/bash -o pipefail
SRC = $(shell find pkg -name "*.go")

VERSION_PACKAGE = github.com/replicatedhq/kustomize-demo-api/pkg/version
VERSION ?=`git describe --tags`
DATE=`date -u +"%Y-%m-%dT%H:%M:%SZ"`

GIT_TREE = $(shell git rev-parse --is-inside-work-tree 2>/dev/null)
ifneq "$(GIT_TREE)" ""
define GIT_UPDATE_INDEX_CMD
git update-index --assume-unchanged
endef
define GIT_SHA
`git rev-parse HEAD`
endef
else
define GIT_UPDATE_INDEX_CMD
echo "Not a git repo, skipping git update-index"
endef
define GIT_SHA
""
endef
endif

define LDFLAGS
-ldflags "\
	-X ${VERSION_PACKAGE}.version=${VERSION} \
	-X ${VERSION_PACKAGE}.gitSHA=${GIT_SHA} \
	-X ${VERSION_PACKAGE}.buildTime=${DATE} \
"
endef

.state/lint-deps: .deps/get_lint_deps.sh
	time ./.deps/get_lint_deps.sh
	@mkdir -p .state/
	@touch .state/lint-deps

.PHONY: lint
lint: .state/lint-deps
	golangci-lint run ./pkg/...
	ineffassign ./pkg

.PHONY: test
test: lint
	go test -v ./pkg/...

bin/kustomize-demo-api: $(SRC)
	go build \
		-mod vendor \
		${LDFLAGS} \
		-i \
		-o bin/kustomize-demo-api \
		.
	@echo built bin/kustomize-demo-api

.PHONY: build
build: bin/kustomize-demo-api

.PHONY: run-docker
run-docker:
	docker build  -t kustomize-demo-api:testing .
	docker run -it -p 3000:3000 kustomize-demo-api:testing
