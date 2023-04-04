FROM golang:1.19-alpine AS builder

ADD . /go/src/github.com/replicatedhq/kustomize-demo-api

WORKDIR /go/src/github.com/replicatedhq/kustomize-demo-api

ARG VERSION=undefined
ARG GITSHA=undefined
ARG BUILDTIME=undefined

ENV CGO_ENABLED=0

RUN go build -mod vendor \
	-ldflags "-X github.com/replicatedhq/kustomize-demo-api/pkg/version.version=$VERSION -X github.com/replicatedhq/kustomize-demo-api/pkg/version.gitSHA=$GITSHA -X github.com/replicatedhq/kustomize-demo-api/pkg/version.buildTime=$BUILDTIME" \
	-i \
	-o /kustomize-demo-api .

FROM alpine:latest
COPY --from=builder /kustomize-demo-api /kustomize-demo-api
EXPOSE 3000
ENTRYPOINT ["/kustomize-demo-api"]
