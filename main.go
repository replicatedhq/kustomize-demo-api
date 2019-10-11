package main

import (
	"github.com/replicatedhq/kustomize-demo-api/pkg/daemon"
)

func main() {
	daemon.Serve()
	return
}
