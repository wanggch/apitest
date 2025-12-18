package yaml

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// Unmarshal parses a very small subset of YAML sufficient for plan files.
func Unmarshal(data []byte, out interface{}) error {
	lines := strings.Split(string(data), "\n")
	value, _, err := parseBlock(lines, 0, 0)
	if err != nil {
		return err
	}
	if value == nil {
		return nil
	}
	b, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, out)
}

// Marshal is not implemented for simplicity.
func Marshal(v interface{}) ([]byte, error) {
	return nil, fmt.Errorf("yaml.Marshal not implemented")
}

type parseResult struct {
	value interface{}
	next  int
}

func parseBlock(lines []string, start, indent int) (interface{}, int, error) {
	var mode string
	obj := map[string]interface{}{}
	arr := []interface{}{}
	i := start
	for i < len(lines) {
		line := lines[i]
		if strings.TrimSpace(line) == "" {
			i++
			continue
		}
		if strings.HasPrefix(strings.TrimSpace(line), "#") {
			i++
			continue
		}
		curIndent := countIndent(line)
		if curIndent < indent {
			break
		}
		if curIndent > indent {
			return nil, i, fmt.Errorf("invalid indentation on line %d", i+1)
		}
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "-") {
			if mode == "map" {
				return nil, i, fmt.Errorf("mixing list and map not supported")
			}
			mode = "list"
			itemVal := strings.TrimSpace(strings.TrimPrefix(trimmed, "-"))
			i++
			var val interface{}
			var err error
			if itemVal == "" {
				val, i, err = parseBlock(lines, i, indent+2)
			} else {
				if strings.Contains(itemVal, ":") {
					synthetic := strings.Repeat(" ", indent+2) + itemVal
					j := i
					for j < len(lines) {
						ln := lines[j]
						if strings.TrimSpace(ln) == "" || strings.HasPrefix(strings.TrimSpace(ln), "#") {
							j++
							continue
						}
						if countIndent(ln) < indent+2 {
							break
						}
						j++
					}
					combined := append([]string{synthetic}, lines[i:j]...)
					val, _, err = parseBlock(combined, 0, indent+2)
					i = j
				} else {
					val = parseScalar(itemVal)
				}
			}
			if err != nil {
				return nil, i, err
			}
			arr = append(arr, val)
			continue
		}

		// map entry
		if mode == "list" {
			return nil, i, fmt.Errorf("mixing list and map not supported")
		}
		mode = "map"
		parts := strings.SplitN(trimmed, ":", 2)
		if len(parts) != 2 {
			return nil, i, fmt.Errorf("invalid line %d", i+1)
		}
		key := strings.TrimSpace(parts[0])
		valText := strings.TrimSpace(parts[1])
		i++
		if valText == "" {
			var err error
			obj[key], i, err = parseBlock(lines, i, indent+2)
			if err != nil {
				return nil, i, err
			}
		} else {
			obj[key] = parseScalar(valText)
		}
	}

	if mode == "list" {
		return arr, i, nil
	}
	return obj, i, nil
}

func parseScalar(val string) interface{} {
	if strings.HasPrefix(val, "\"") && strings.HasSuffix(val, "\"") {
		return strings.Trim(val, "\"")
	}
	if strings.HasPrefix(val, "'") && strings.HasSuffix(val, "'") {
		return strings.Trim(val, "'")
	}
	if i, err := strconv.ParseInt(val, 10, 64); err == nil {
		return i
	}
	if f, err := strconv.ParseFloat(val, 64); err == nil {
		return f
	}
	lower := strings.ToLower(val)
	if lower == "true" {
		return true
	}
	if lower == "false" {
		return false
	}
	if lower == "null" || lower == "~" {
		return nil
	}
	return val
}

func countIndent(s string) int {
	count := 0
	for _, ch := range s {
		if ch == ' ' {
			count++
		} else {
			break
		}
	}
	return count
}
