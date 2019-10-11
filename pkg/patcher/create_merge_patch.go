package patcher

import (
	"encoding/json"

	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
	"github.com/replicatedhq/ship/pkg/util"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/client-go/kubernetes/scheme"
)

func CreateTwoWayMergePatch(original, modified []byte) ([]byte, error) {
	originalJSON, err := yaml.YAMLToJSON(original)
	if err != nil {
		return nil, errors.Wrap(err, "convert original file to json")
	}

	modifiedJSON, err := yaml.YAMLToJSON(modified)
	if err != nil {
		return nil, errors.Wrap(err, "convert modified file to json")
	}

	r, err := util.NewKubernetesResource(originalJSON)
	if err != nil {
		return nil, errors.Wrap(err, "create kube resource with original json")
	}

	versionedObj, err := scheme.Scheme.New(util.ToGroupVersionKind(r.Id().Gvk()))
	if err != nil {
		return nil, errors.Wrap(err, "read group, version kind from kube resource")
	}

	patchBytes, err := strategicpatch.CreateTwoWayMergePatch(originalJSON, modifiedJSON, versionedObj)
	if err != nil {
		return nil, errors.Wrap(err, "create two way merge patch")
	}

	modifiedPatchJSON, err := writeHeaderToPatch(originalJSON, patchBytes)
	if err != nil {
		return nil, errors.Wrap(err, "write original header to patch")
	}

	patch, err := yaml.JSONToYAML(modifiedPatchJSON)
	if err != nil {
		return nil, errors.Wrap(err, "convert merge patch json to yaml")
	}

	return patch, nil
}

func writeHeaderToPatch(originalJSON, patchJSON []byte) ([]byte, error) {
	original := map[string]interface{}{}
	patch := map[string]interface{}{}

	err := json.Unmarshal(originalJSON, &original)
	if err != nil {
		return nil, errors.Wrap(err, "unmarshal original json")
	}

	err = json.Unmarshal(patchJSON, &patch)
	if err != nil {
		return nil, errors.Wrap(err, "unmarshal patch json")
	}

	originalAPIVersion, ok := original["apiVersion"]
	if !ok {
		return nil, errors.New("no apiVersion key present in original")
	}

	originalKind, ok := original["kind"]
	if !ok {
		return nil, errors.New("no kind key present in original")
	}

	originalMetadata, ok := original["metadata"]
	if !ok {
		return nil, errors.New("no metadata key present in original")
	}

	patch["apiVersion"] = originalAPIVersion
	patch["kind"] = originalKind
	patch["metadata"] = originalMetadata

	modifiedPatch, err := json.Marshal(patch)
	if err != nil {
		return nil, errors.Wrap(err, "marshal modified patch json")
	}

	return modifiedPatch, nil
}

// given a base yaml and an array of patches for that yaml, return a single unified patch that produces the same output
// as applying each patch in series would have
func CombinePatches(original []byte, patches [][]byte) ([]byte, error) {
	desiredOutput := original
	for idx, patch := range patches {
		var err error
		desiredOutput, err = ApplyPatch(desiredOutput, patch)
		if err != nil {
			return nil, errors.Wrapf(err, "apply patch #%d", idx)
		}
	}

	return CreateTwoWayMergePatch(original, desiredOutput)
}
