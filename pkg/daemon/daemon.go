package daemon

import (
	"fmt"
	"os"
	"strconv"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/replicatedhq/ship/pkg/util"

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
	kust.POST("generate-base", KustomizeGenerateBase)
	kust.POST("generate-overlay", KustomizeGenerateOverlay)

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

type localKustomization struct {
	APIVersion            string   `json:"apiVersion,omitempty" yaml:"apiVersion,omitempty"`
	Kind                  string   `json:"kind,omitempty" yaml:"kind,omitempty"`
	Bases                 []string `json:"bases,omitempty" yaml:"bases,omitempty"`
	Resources             []string `json:"resources,omitempty" yaml:"resources,omitempty"`
	PatchesStrategicMerge []string `json:"patchesStrategicMerge,omitempty" yaml:"patchesStrategicMerge,omitempty"`
}

func renderKustomize(resources, patches, bases []string) ([]byte, error) {
	genKust := localKustomization{
		Kind:                  "Kustomization",
		APIVersion:            "kustomize.config.k8s.io/v1beta1",
		Resources:             resources,
		PatchesStrategicMerge: patches,
		Bases:                 bases,
	}

	kustBytes, err := util.MarshalIndent(2, genKust)
	if err != nil {
		return nil, err
	}
	return kustBytes, nil
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

	kustBytes, err := renderKustomize(request.Resources, request.Patches, request.Bases)
	if err != nil {
		c.AbortWithError(500, err)
		return
	}

	c.Header("Access-Control-Allow-Origin", "*")
	c.IndentedJSON(200, map[string]interface{}{
		"kustomization": string(kustBytes),
	})
}

func KustomizeGenerateBase(c *gin.Context) {
	type Request struct {
		Resources []string `json:"resources"`
	}
	var request Request

	err := c.BindJSON(&request)
	if err != nil {
		c.AbortWithError(500, err)
		return
	}

	kustBytes, err := renderKustomize(request.Resources, nil, nil)
	if err != nil {
		c.AbortWithError(500, err)
		return
	}

	c.Header("Access-Control-Allow-Origin", "*")
	c.IndentedJSON(200, map[string]interface{}{
		"kustomization": string(kustBytes),
	})
}

func KustomizeGenerateOverlay(c *gin.Context) {
	type Request struct {
		Patches []string `json:"patches"`
	}
	var request Request

	err := c.BindJSON(&request)
	if err != nil {
		c.AbortWithError(500, err)
		return
	}

	kustBytes, err := renderKustomize(nil, request.Patches, []string{"../base"})
	if err != nil {
		c.AbortWithError(500, err)
		return
	}

	c.Header("Access-Control-Allow-Origin", "*")
	c.IndentedJSON(200, map[string]interface{}{
		"kustomization": string(kustBytes),
	})
}
