package patcher

import (
	"path/filepath"

	"github.com/pkg/errors"
	"sigs.k8s.io/kustomize/k8sdeps"
	"sigs.k8s.io/kustomize/pkg/fs"
	"sigs.k8s.io/kustomize/pkg/loader"
	"sigs.k8s.io/kustomize/pkg/target"
)

const basicK8sYaml = `
kind: ""
apiversion: ""
patchesStrategicMerge:
- patch.yaml
resources:
- original.yaml
`

func ApplyPatch(original []byte, patch []byte) ([]byte, error) {
	fileSys := fs.MakeFakeFS()
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
	_, err = k8sFile.Write([]byte(basicK8sYaml))
	if err != nil {
		return nil, errors.Wrap(err, "write kustomization.yaml")
	}

	applied, err := runKustomize(fileSys, "/")
	return applied, err
}

func runKustomize(fSys fs.FileSystem, kustomizationPath string) ([]byte, error) {
	absPath, err := filepath.Abs(kustomizationPath)
	if err != nil {
		return nil, err
	}

	ldr, err := loader.NewLoader(absPath, fSys)
	if err != nil {
		return nil, errors.Wrap(err, "make loader")
	}

	k8sFactory := k8sdeps.NewFactory()

	kt, err := target.NewKustTarget(ldr, k8sFactory.ResmapF, k8sFactory.TransformerF)
	if err != nil {
		return nil, errors.Wrap(err, "make customized kustomize target")
	}

	allResources, err := kt.MakeCustomizedResMap()
	if err != nil {
		return nil, errors.Wrap(err, "make customized res map")
	}

	// Output the objects.
	res, err := allResources.EncodeAsYaml()
	if err != nil {
		return nil, errors.Wrap(err, "encode as yaml")
	}
	return res, err
}
