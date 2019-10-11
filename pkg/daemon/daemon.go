package daemon

import (
	"context"

	"github.com/gin-gonic/gin"

	"github.com/replicatedhq/kustomize-demo-api/pkg/version"
)

func Serve(ctx context.Context) error {
	g := gin.New()

	root := g.Group("/")
	root.GET("/healthz", Healthz)
	root.GET("/livez", Healthz)

	kust := root.Group("kustomize")
	kust.POST("patch", KustomizePatch)

	return g.Run(":3000")
}

func Healthz(c *gin.Context) {
	c.JSON(200, map[string]interface{}{
		"version":   version.Version(),
		"sha":       version.GitSHA(),
		"buildTime": version.BuildTime(),
	})
}

func KustomizePatch(c *gin.Context) {
	c.String(200, "not yet implemented")
}
