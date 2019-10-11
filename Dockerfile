FROM golang:1.13-alpine

ADD . /go/src/github.com/replicatedhq/kustomize-demo-api

WORKDIR /go/src/github.com/replicatedhq/kustomize-demo-api

RUN go build -mod vendor -o /kustomize-demo-api .

FROM alpine:latest
COPY --from=0 /kustomize-demo-api /kustomize-demo-api
EXPOSE 3000
ENTRYPOINT ["/kustomize-demo-api"]
