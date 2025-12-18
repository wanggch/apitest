package templ

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var (
	// ErrMissingVar indicates template variable not found.
	ErrMissingVar = errors.New("missing variable")
)

var templatePattern = regexp.MustCompile(`{{\s*([\w\.-]+)\s*}}`)

// ApplyString replaces template variables in a string using provided context map.
func ApplyString(input string, ctx map[string]string) (string, error) {
	var missing []string
	out := templatePattern.ReplaceAllStringFunc(input, func(m string) string {
		matches := templatePattern.FindStringSubmatch(m)
		key := matches[1]
		val, ok := ctx[key]
		if !ok {
			missing = append(missing, key)
			return m
		}
		return val
	})
	if len(missing) > 0 {
		return out, fmt.Errorf("%w: %s", ErrMissingVar, strings.Join(missing, ","))
	}
	return out, nil
}

// ApplyInterface walks through interface{} and applies template replacement on strings.
func ApplyInterface(data interface{}, ctx map[string]string) (interface{}, error) {
	switch v := data.(type) {
	case string:
		return ApplyString(v, ctx)
	case []interface{}:
		var arr []interface{}
		for _, item := range v {
			replaced, err := ApplyInterface(item, ctx)
			if err != nil {
				return nil, err
			}
			arr = append(arr, replaced)
		}
		return arr, nil
	case map[string]interface{}:
		res := make(map[string]interface{})
		for k, val := range v {
			replaced, err := ApplyInterface(val, ctx)
			if err != nil {
				return nil, err
			}
			res[k] = replaced
		}
		return res, nil
	default:
		return v, nil
	}
}

// MergeContexts merges multiple maps with later maps overriding earlier ones.
func MergeContexts(maps ...map[string]string) map[string]string {
	out := make(map[string]string)
	for _, m := range maps {
		for k, v := range m {
			out[k] = v
		}
	}
	return out
}
