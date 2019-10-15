package patcher

import (
	"path/filepath"

	"github.com/pkg/errors"
	"sigs.k8s.io/kustomize/v3/k8sdeps/kunstruct"
	"sigs.k8s.io/kustomize/v3/k8sdeps/transformer"
	"sigs.k8s.io/kustomize/v3/k8sdeps/validator"
	"sigs.k8s.io/kustomize/v3/pkg/fs"
	"sigs.k8s.io/kustomize/v3/pkg/loader"
	"sigs.k8s.io/kustomize/v3/pkg/plugins"
	"sigs.k8s.io/kustomize/v3/pkg/resmap"
	"sigs.k8s.io/kustomize/v3/pkg/resource"
	"sigs.k8s.io/kustomize/v3/pkg/target"
)

const basicK8sYaml = `
kind: ""
apiversion: ""
patchesStrategicMerge:
- patch.yaml
resources:
- original.yaml
`

const noPatchYaml = `
kind: ""
apiversion: ""
resources:
- original.yaml
`

func ApplyPatch(original []byte, patch []byte) ([]byte, error) {
	fileSys := fs.MakeFsInMemory()
	originalFile, err := fileSys.Create("/original.yaml")
	if err != nil {
		return nil, errors.Wrap(err, "create original.yaml")
	}
	_, err = originalFile.Write(original)
	if err != nil {
		return nil, errors.Wrap(err, "write original.yaml")
	}

	patchFile, err := fileSys.Create("/patch.yaml")
	if err != nil {
		return nil, errors.Wrap(err, "create patch.yaml")
	}
	_, err = patchFile.Write(patch)
	if err != nil {
		return nil, errors.Wrap(err, "write patch.yaml")
	}

	k8sFile, err := fileSys.Create("/kustomization.yaml")
	if err != nil {
		return nil, errors.Wrap(err, "create kustomization.yaml")
	}

	if len(patch) != 0 {
		_, err = k8sFile.Write([]byte(basicK8sYaml))
		if err != nil {
			return nil, errors.Wrap(err, "write kustomization.yaml")
		}
	} else {
		_, err = k8sFile.Write([]byte(noPatchYaml))
		if err != nil {
			return nil, errors.Wrap(err, "write kustomization.yaml")
		}
	}

	applied, err := runKustomize(fileSys, "/")
	return applied, err
}

func runKustomize(fSys fs.FileSystem, kustomizationPath string) ([]byte, error) {
	absPath, err := filepath.Abs(kustomizationPath)
	if err != nil {
		return nil, err
	}

	ldr, err := loader.NewLoader(loader.RestrictionRootOnly, &validator.KustValidator{}, absPath, fSys)
	if err != nil {
		return nil, errors.Wrap(err, "make loader")
	}

	rf := resmap.NewFactory(resource.NewFactory(kunstruct.NewKunstructuredFactoryImpl()), transformer.NewFactoryImpl())
	pc := plugins.DefaultPluginConfig()
	kt, err := target.NewKustTarget(ldr, rf, transformer.NewFactoryImpl(), plugins.NewLoader(pc, rf))
	if err != nil {
		return nil, errors.Wrap(err, "make customized kustomize target")
	}

	allResources, err := kt.MakeCustomizedResMap()
	if err != nil {
		return nil, errors.Wrap(err, "make customized res map")
	}

	// Output the objects.
	res, err := allResources.AsYaml()
	if err != nil {
		return nil, errors.Wrap(err, "encode as yaml")
	}
	return res, err
}
