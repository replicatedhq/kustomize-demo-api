package main

import (
	"context"

	"github.com/replicatedhq/kustomize-demo-api/pkg/daemon"
)

func main() {
	daemon.Serve(context.Background())
	return
}
