package gjson

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

type Type int

const (
	Null Type = iota
	False
	Number
	String
	JSON
	True
)

type Result struct {
	value interface{}
	Type  Type
}

func (r Result) Exists() bool {
	return r.value != nil
}

func (r Result) Value() interface{} { return r.value }

func (r Result) String() string {
	switch v := r.value.(type) {
	case string:
		return v
	case json.Number:
		return v.String()
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(v)
	default:
		return fmt.Sprint(v)
	}
}

func (r Result) Num() float64 {
	switch v := r.value.(type) {
	case json.Number:
		f, _ := v.Float64()
		return f
	case float64:
		return v
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case string:
		f, _ := strconv.ParseFloat(v, 64)
		return f
	default:
		return 0
	}
}

func (r Result) Bool() bool {
	switch v := r.value.(type) {
	case bool:
		return v
	case string:
		b, _ := strconv.ParseBool(v)
		return b
	default:
		return false
	}
}

func Valid(data string) bool {
	var v interface{}
	return json.Unmarshal([]byte(data), &v) == nil
}

func Parse(data string) Result {
	var v interface{}
	if err := json.Unmarshal([]byte(data), &v); err != nil {
		return Result{Type: Null}
	}
	return Result{value: v, Type: detectType(v)}
}

func Get(data, path string) Result {
	root := Parse(data)
	if root.value == nil {
		return Result{Type: Null}
	}
	parts := splitPath(path)
	current := root.value
	for _, p := range parts {
		switch node := current.(type) {
		case map[string]interface{}:
			val, ok := node[p.name]
			if !ok {
				return Result{Type: Null}
			}
			current = selectIndex(val, p.index)
		case []interface{}:
			if p.name != "" {
				return Result{Type: Null}
			}
			current = selectIndex(node, p.index)
		default:
			return Result{Type: Null}
		}
	}
	return Result{value: current, Type: detectType(current)}
}

// Get navigates within an existing result using a path.
func (r Result) Get(path string) Result {
	if r.value == nil {
		return Result{Type: Null}
	}
	parts := splitPath(path)
	current := r.value
	for _, p := range parts {
		switch node := current.(type) {
		case map[string]interface{}:
			val, ok := node[p.name]
			if !ok {
				return Result{Type: Null}
			}
			current = selectIndex(val, p.index)
		case []interface{}:
			if p.name != "" {
				return Result{Type: Null}
			}
			current = selectIndex(node, p.index)
		default:
			return Result{Type: Null}
		}
	}
	return Result{value: current, Type: detectType(current)}
}

type pathPart struct {
	name  string
	index *int
}

func splitPath(path string) []pathPart {
	if path == "" {
		return nil
	}
	rawParts := strings.Split(path, ".")
	parts := make([]pathPart, 0, len(rawParts))
	for _, rp := range rawParts {
		if rp == "" {
			continue
		}
		part := pathPart{name: rp}
		if strings.Contains(rp, "[") && strings.HasSuffix(rp, "]") {
			name := rp[:strings.Index(rp, "[")]
			idxStr := rp[strings.Index(rp, "[")+1 : len(rp)-1]
			if idxStr != "" {
				if idx, err := strconv.Atoi(idxStr); err == nil {
					part.name = name
					part.index = &idx
				}
			}
		}
		parts = append(parts, part)
	}
	return parts
}

func selectIndex(value interface{}, idx *int) interface{} {
	if idx == nil {
		return value
	}
	switch arr := value.(type) {
	case []interface{}:
		if *idx >= 0 && *idx < len(arr) {
			return arr[*idx]
		}
	}
	return nil
}

func detectType(v interface{}) Type {
	switch v := v.(type) {
	case nil:
		return Null
	case bool:
		if v {
			return True
		}
		return False
	case json.Number, float64, int, int64:
		return Number
	case string:
		return String
	case map[string]interface{}, []interface{}:
		return JSON
	default:
		return JSON
	}
}
