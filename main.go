package main

import (
	"fmt"

	"github.com/replicatedhq/kustomize-demo-api/pkg/daemon"
	"github.com/replicatedhq/kustomize-demo-api/pkg/version"
)

func main() {
	fmt.Printf("running kustomize-demo-api\n%+v\n", version.GetBuild())
	daemon.Serve()
	return
}
