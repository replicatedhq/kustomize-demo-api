package daemon

import (
	"fmt"
	"strconv"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"

	"github.com/replicatedhq/kustomize-demo-api/pkg/patcher"
	"github.com/replicatedhq/kustomize-demo-api/pkg/version"
)

func Serve() error {
	g := gin.New()
	g.Use(
		gin.LoggerWithWriter(gin.DefaultWriter, "/healthz", "/livez"),
		gin.Recovery(),
		cors.Default(),
	)

	root := g.Group("/")
	root.GET("/healthz", Healthz)
	root.GET("/livez", Healthz)

	kust := root.Group("kustomize")
	kust.POST("patch", KustomizePatch)
	kust.POST("apply", KustomizeApply)

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
		Patch    string        `json:"existing_patch"`
		Path     []interface{} `json:"path"`
	}
	var request Request

	err := c.BindJSON(&request)
	if err != nil {
		c.AbortWithError(500, err)
		return
	}

	var stringPath []string
	for idx, value := range request.Path {
		switch value := value.(type) {
		case float64:
			stringPath = append(stringPath, strconv.FormatFloat(value, 'f', 0, 64))
		case string:
			stringPath = append(stringPath, value)
		default:
			c.AbortWithError(500, fmt.Errorf("value %+v at idx %d of path is not a string or a float", value, idx))
			return
		}
	}

	modified, err := patcher.ModifyField([]byte(request.Original), stringPath)
	if err != nil {
		c.AbortWithError(500, errors.Wrapf(err, "modify field"))
		return
	}

	patch, err := patcher.CreateTwoWayMergePatch([]byte(request.Original), modified)
	if err != nil {
		c.AbortWithError(500, errors.Wrapf(err, "create patch for field"))
		return
	}

	if request.Patch != "" {
		patch, err = patcher.CombinePatches([]byte(request.Original), [][]byte{[]byte(request.Patch), patch})
		if err != nil {
			c.AbortWithError(500, errors.Wrapf(err, "merge new patch with existing"))
			return
		}
	}

	c.JSON(200, map[string]interface{}{
		"patch": string(patch),
	})
}

func KustomizeApply(c *gin.Context) {
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
		c.AbortWithError(500, errors.Wrapf(err, "apply patch"))
		return
	}

	c.JSON(200, map[string]interface{}{
		"modified": string(modified),
	})
}
