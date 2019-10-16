package daemon

import (
	"fmt"
	"os"
	"strconv"

	"github.com/ghodss/yaml"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"sigs.k8s.io/kustomize/v3/pkg/types"

	"github.com/replicatedhq/kustomize-demo-api/pkg/patcher"
	"github.com/replicatedhq/kustomize-demo-api/pkg/version"
)

func setupRouter() *gin.Engine {
	g := gin.New()
	g.Use(
		gin.LoggerWithWriter(gin.DefaultWriter, "/healthz", "/livez"),
		gin.Recovery(),
	)

	corsConfig := cors.DefaultConfig()
	corsConfig.AllowAllOrigins = true
	corsConfig.AllowHeaders = []string{"*"}
	g.Use(cors.New(corsConfig))

	root := g.Group("/")
	root.GET("/healthz", Healthz)
	root.GET("/livez", Healthz)

	kust := root.Group("kustomize")
	kust.POST("patch", KustomizePatch)
	kust.POST("apply", KustomizeApply)
	kust.POST("generate", KustomizeGenerate)

	return g
}

func Serve() error {
	g := setupRouter()

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	return g.Run(":" + port)
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

	// this code - replacing the original yaml with the patched yaml before generating new patches - fixes issues
	// related modifying elements in a list that are already impacted by the existing patch
	var baseYaml = []byte(request.Original)
	if request.Patch != "" {
		patchedYaml, err := patcher.ApplyPatch([]byte(request.Original), []byte(request.Patch))
		if err != nil {
			c.AbortWithError(500, errors.Wrapf(err, "apply existing patch"))
			return
		}
		baseYaml = patchedYaml
	}

	modified, err := patcher.ModifyField(baseYaml, stringPath)
	if err != nil {
		c.AbortWithError(500, errors.Wrapf(err, "modify field"))
		return
	}

	patch, err := patcher.CreateTwoWayMergePatch(baseYaml, modified)
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

	c.Header("Access-Control-Allow-Origin", "*")
	c.IndentedJSON(200, map[string]interface{}{
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

	c.Header("Access-Control-Allow-Origin", "*")
	c.IndentedJSON(200, map[string]interface{}{
		"modified": string(modified),
	})
}

func KustomizeGenerate(c *gin.Context) {
	type Request struct {
		Resources []string `json:"resources"`
		Patches   []string `json:"patches"`
		Bases     []string `json:"bases"`
	}
	var request Request

	err := c.BindJSON(&request)
	if err != nil {
		c.AbortWithError(500, err)
		return
	}

	patches := []types.PatchStrategicMerge{}
	for _, patchPath := range request.Patches {
		patches = append(patches, types.PatchStrategicMerge(patchPath))
	}

	genKust := types.Kustomization{
		TypeMeta: types.TypeMeta{
			Kind:       "Kustomization",
			APIVersion: "kustomize.config.k8s.io/v1beta1",
		},
		Resources:             request.Resources,
		PatchesStrategicMerge: patches,
		Bases:                 request.Bases,
	}

	kustBytes, err := yaml.Marshal(genKust)
	if err != nil {
		c.AbortWithError(500, err)
		return
	}

	c.Header("Access-Control-Allow-Origin", "*")
	c.IndentedJSON(200, map[string]interface{}{
		"kustomization": string(kustBytes),
	})
}
