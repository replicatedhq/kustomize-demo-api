package patcher

import (
	"reflect"
	"strconv"

	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
)

const PATCH_TOKEN = "TO_BE_MODIFIED"

func ModifyField(original []byte, path []string) ([]byte, error) {
	originalMap := map[string]interface{}{}

	err := yaml.Unmarshal(original, &originalMap)
	if err != nil {
		return nil, errors.Wrap(err, "unmarshal yaml")
	}

	modified, err := modifyField(originalMap, []string{}, path)
	if err != nil {
		return nil, errors.Wrap(err, "error modifying value")
	}

	modifiedYAML, err := yaml.Marshal(modified)
	if err != nil {
		return nil, errors.Wrap(err, "marshal yaml")
	}

	return modifiedYAML, nil
}

func modifyField(original interface{}, current []string, path []string) (interface{}, error) {
	originalType := reflect.TypeOf(original)
	if original == nil {
		return nil, nil
	}
	switch originalType.Kind() {
	case reflect.Map:
		typedOriginal, ok := original.(map[string]interface{})
		modifiedMap := make(map[string]interface{})
		if !ok {
			return nil, errors.New("error asserting map")
		}
		for key, value := range typedOriginal {
			modifiedValue, err := modifyField(value, append(current, key), path)
			if err != nil {
				return nil, err
			}
			modifiedMap[key] = modifiedValue
		}
		return modifiedMap, nil
	case reflect.Slice:
		typedOriginal, ok := original.([]interface{})
		modifiedSlice := make([]interface{}, len(typedOriginal))
		if !ok {
			return nil, errors.New("error asserting slice")
		}
		for key, value := range typedOriginal {
			modifiedValue, err := modifyField(value, append(current, strconv.Itoa(key)), path)
			if err != nil {
				return nil, err
			}
			modifiedSlice[key] = modifiedValue
		}
		return modifiedSlice, nil
	default:
		for idx := range path {
			if current[idx] != path[idx] {
				return original, nil
			}
		}
		return PATCH_TOKEN, nil
	}
}
