package daemon

import (
	"context"
	"errors"

	"github.com/gin-gonic/gin"

	"github.com/replicatedhq/kustomize-demo-api/pkg/patcher"
	"github.com/replicatedhq/kustomize-demo-api/pkg/version"
)

func Serve(ctx context.Context) error {
	g := gin.New()

	root := g.Group("/")
	root.GET("/healthz", Healthz)
	root.GET("/livez", Healthz)

	kust := root.Group("kustomize")
	kust.POST("patch", KustomizePatch)
	kust.POST("apply", ApplyPatch)

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
	type Request struct {
		Original string        `json:"original"`
		Current  string        `json:"current"`
		Path     []interface{} `json:"path"`
		Resource string        `json:"resource"`
	}
	var request Request

	err := c.BindJSON(&request)
	if err != nil {
		c.AbortWithError(500, err)
		return
	}

	c.String(200, "not yet implemented")
}

func ApplyPatch(c *gin.Context) {
	type Request struct {
		Resource string `json:"resource"`
		Patch    string `json:"patch"`
	}
	var request Request

	err := c.BindJSON(&request)
	if err != nil {
		c.AbortWithError(500, err)
		return
	}

	modified, err := patcher.ApplyPatch([]byte(request.Resource), []byte(request.Patch))
	if err != nil {
		c.AbortWithError(500, errors.New("internal_server_error"))
		return
	}

	c.JSON(200, map[string]interface{}{
		"modified": string(modified),
	})
}
