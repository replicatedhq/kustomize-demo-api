SHELL := /bin/bash -o pipefail
PROJECT_NAME ?= kustomize-demo-api

SRC = $(shell find pkg -name "*.go")

VERSION_PACKAGE = github.com/replicatedhq/kustomize-demo-api/pkg/version
VERSION ?=`git describe --tags`
DATE=`date -u +"%Y-%m-%dT%H:%M:%SZ"`

export GO111MODULE=on

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

.PHONY: build-docker
build-docker:
	docker build  -t ${DOCKERNAME} --build-arg VERSION=${VERSION} --build-arg GITSHA=${GIT_SHA} --build-arg BUILDTIME=${DATE} .

.PHONY: run-docker
run-docker: DOCKERNAME = kustomize-demo-api:testing
run-docker: build-docker
	docker run -it -p 3000:3000 kustomize-demo-api:testing



# CI deployment tools

.PHONY: build-staging
build-staging: REGISTRY = 923411875752.dkr.ecr.us-east-1.amazonaws.com
build-staging: push-docker

.PHONY: build-production
build-production: REGISTRY = 799720048698.dkr.ecr.us-east-1.amazonaws.com
build-production: push-docker

.PHONY: push-docker
push-docker: DOCKERNAME = ${PROJECT_NAME}:push-docker
push-docker: $(SRC) build-docker
	docker tag ${DOCKERNAME} $(REGISTRY)/${PROJECT_NAME}:$${BUILDKITE_COMMIT:0:7}
	docker push $(REGISTRY)/${PROJECT_NAME}:$${BUILDKITE_COMMIT:0:7}

.PHONY: publish-staging
publish-staging: REGISTRY = 923411875752.dkr.ecr.us-east-1.amazonaws.com
publish-staging: BUILD_VERSION = $(shell echo ${BUILDKITE_COMMIT} | cut -c1-7)
publish-staging: OVERLAY = staging
publish-staging: GITOPS_OWNER = replicatedcom
publish-staging: GITOPS_REPO = gitops-deploy
publish-staging: GITOPS_BRANCH = master
publish-staging: gitops

.PHONY: publish-production
publish-production: REGISTRY = 799720048698.dkr.ecr.us-east-1.amazonaws.com
publish-production: BUILD_VERSION = $(shell echo ${BUILDKITE_COMMIT} | cut -c1-7)
publish-production: OVERLAY = production
publish-production: GITOPS_OWNER = replicatedcom
publish-production: GITOPS_REPO = gitops-deploy
publish-production: GITOPS_BRANCH = release
publish-production: gitops

.PHONY: gitops
gitops:
	cd kustomize/overlays/$(OVERLAY); kustomize edit set image $(REGISTRY)/${PROJECT_NAME}=$(REGISTRY)/${PROJECT_NAME}:${BUILD_VERSION}

	rm -rf deploy/$(OVERLAY)/work
	mkdir -p deploy/$(OVERLAY)/work; cd deploy/$(OVERLAY)/work; git clone --single-branch -b $(GITOPS_BRANCH) git@github.com:$(GITOPS_OWNER)/$(GITOPS_REPO)
	mkdir -p deploy/$(OVERLAY)/work/$(GITOPS_REPO)/${PROJECT_NAME}

	kustomize build kustomize/overlays/$(OVERLAY) > deploy/$(OVERLAY)/work/$(GITOPS_REPO)/${PROJECT_NAME}/${PROJECT_NAME}.yaml

	cd deploy/$(OVERLAY)/work/$(GITOPS_REPO)/${PROJECT_NAME}; \
	  git add . ;\
	  git commit --allow-empty -m "$${BUILDKITE_BUILD_URL}"; \
			git push origin $(GITOPS_BRANCH)
