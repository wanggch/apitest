package assert

import (
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/tidwall/gjson"

	"apitest/internal/config"
	"apitest/internal/templ"
)

// Result describes single assertion result.
type Result struct {
	Pass    bool
	Message string
}

// Evaluate executes assertions against response.
func Evaluate(assertions []config.Assertion, respBody string, headers http.Header, status int, ctx map[string]string) []Result {
	results := make([]Result, 0, len(assertions))
	for _, a := range assertions {
		r := evaluateOne(a, respBody, headers, status, ctx)
		results = append(results, r)
		if !r.Pass {
			break
		}
	}
	return results
}

func evaluateOne(a config.Assertion, body string, headers http.Header, status int, ctx map[string]string) Result {
	switch strings.ToLower(a.Type) {
	case "status":
		return assertStatus(a, status)
	case "header":
		return assertHeader(a, headers)
	case "body":
		return assertBody(a, body)
	case "json":
		return assertJSON(a, body, ctx)
	default:
		return Result{Pass: false, Message: fmt.Sprintf("unknown assertion type %s", a.Type)}
	}
}

func assertStatus(a config.Assertion, status int) Result {
	expect := toFloat(a.Expect)
	switch a.Op {
	case "==":
		if float64(status) == expect {
			return Result{Pass: true, Message: fmt.Sprintf("status == %v", expect)}
		}
		return Result{Pass: false, Message: fmt.Sprintf("status %d != %v", status, expect)}
	case "!=":
		if float64(status) != expect {
			return Result{Pass: true, Message: fmt.Sprintf("status != %v", expect)}
		}
		return Result{Pass: false, Message: fmt.Sprintf("status %d == %v", status, expect)}
	case ">=":
		if float64(status) >= expect {
			return Result{Pass: true, Message: fmt.Sprintf("status >= %v", expect)}
		}
		return Result{Pass: false, Message: fmt.Sprintf("status %d < %v", status, expect)}
	case "<=":
		if float64(status) <= expect {
			return Result{Pass: true, Message: fmt.Sprintf("status <= %v", expect)}
		}
		return Result{Pass: false, Message: fmt.Sprintf("status %d > %v", status, expect)}
	default:
		return Result{Pass: false, Message: fmt.Sprintf("unknown status op %s", a.Op)}
	}
}

func assertHeader(a config.Assertion, headers http.Header) Result {
	name := a.Path
	if name == "" {
		name = a.Name
	}
	if name == "" {
		return Result{Pass: false, Message: "header name missing"}
	}
	values := headers.Values(name)
	switch a.Op {
	case "exists":
		if len(values) > 0 {
			return Result{Pass: true, Message: fmt.Sprintf("header %s exists", name)}
		}
		return Result{Pass: false, Message: fmt.Sprintf("header %s not found", name)}
	case "contains":
		expect := fmt.Sprint(a.Expect)
		for _, v := range values {
			if strings.Contains(v, expect) {
				return Result{Pass: true, Message: fmt.Sprintf("header %s contains %s", name, expect)}
			}
		}
		return Result{Pass: false, Message: fmt.Sprintf("header %s does not contain %s", name, expect)}
	case "==":
		expect := fmt.Sprint(a.Expect)
		if len(values) > 0 && values[0] == expect {
			return Result{Pass: true, Message: fmt.Sprintf("header %s == %s", name, expect)}
		}
		return Result{Pass: false, Message: fmt.Sprintf("header %s value %v != %s", name, values, expect)}
	case "!=":
		expect := fmt.Sprint(a.Expect)
		if len(values) == 0 || values[0] != expect {
			return Result{Pass: true, Message: fmt.Sprintf("header %s != %s", name, expect)}
		}
		return Result{Pass: false, Message: fmt.Sprintf("header %s equals %s", name, expect)}
	default:
		return Result{Pass: false, Message: fmt.Sprintf("unknown header op %s", a.Op)}
	}
}

func assertBody(a config.Assertion, body string) Result {
	switch a.Op {
	case "contains":
		expect := fmt.Sprint(a.Expect)
		if strings.Contains(body, expect) {
			return Result{Pass: true, Message: fmt.Sprintf("body contains %s", expect)}
		}
		return Result{Pass: false, Message: fmt.Sprintf("body does not contain %s", expect)}
	case "regex":
		pattern := fmt.Sprint(a.Expect)
		re, err := regexp.Compile(pattern)
		if err != nil {
			return Result{Pass: false, Message: fmt.Sprintf("invalid regex: %v", err)}
		}
		if re.MatchString(body) {
			return Result{Pass: true, Message: fmt.Sprintf("body matches %s", pattern)}
		}
		return Result{Pass: false, Message: fmt.Sprintf("body does not match %s", pattern)}
	case "len_gt":
		expect := int(toFloat(a.Expect))
		if len(body) > expect {
			return Result{Pass: true, Message: fmt.Sprintf("body length > %d", expect)}
		}
		return Result{Pass: false, Message: fmt.Sprintf("body length %d <= %d", len(body), expect)}
	case "len_eq":
		expect := int(toFloat(a.Expect))
		if len(body) == expect {
			return Result{Pass: true, Message: fmt.Sprintf("body length == %d", expect)}
		}
		return Result{Pass: false, Message: fmt.Sprintf("body length %d != %d", len(body), expect)}
	default:
		return Result{Pass: false, Message: fmt.Sprintf("unknown body op %s", a.Op)}
	}
}

func assertJSON(a config.Assertion, body string, ctx map[string]string) Result {
	if !gjson.Valid(body) {
		return Result{Pass: false, Message: "response body is not valid JSON"}
	}
	res := gjson.Parse(body).Get(a.Path)
	switch a.Op {
	case "exists":
		if res.Exists() && res.Value() != nil {
			return Result{Pass: true, Message: fmt.Sprintf("json %s exists", a.Path)}
		}
		return Result{Pass: false, Message: fmt.Sprintf("json path %s does not exist", a.Path)}
	case "==", "!=", "contains", "gt", "lt":
		var expectVal interface{} = a.Expect
		if str, ok := a.Expect.(string); ok {
			if strings.Contains(str, "{{") {
				replaced, err := templ.ApplyString(str, ctx)
				if err != nil {
					return Result{Pass: false, Message: fmt.Sprintf("expect template: %v", err)}
				}
				expectVal = replaced
			}
		}
		return compareJSON(a.Op, a.Path, res, expectVal)
	default:
		return Result{Pass: false, Message: fmt.Sprintf("unknown json op %s", a.Op)}
	}
}

func compareJSON(op, path string, val gjson.Result, expect interface{}) Result {
	if op == "contains" {
		expStr := fmt.Sprint(expect)
		actual := val.String()
		if strings.Contains(actual, expStr) {
			return Result{Pass: true, Message: fmt.Sprintf("json %s contains %s", path, expStr)}
		}
		return Result{Pass: false, Message: fmt.Sprintf("json %s value %s does not contain %s", path, actual, expStr)}
	}

	if op == "gt" || op == "lt" {
		if !val.Exists() {
			return Result{Pass: false, Message: fmt.Sprintf("json path %s does not exist", path)}
		}
		if val.Type != gjson.Number {
			return Result{Pass: false, Message: fmt.Sprintf("json %s not a number", path)}
		}
		actualNum := val.Num()
		expNum := toFloat(expect)
		if op == "gt" {
			if actualNum > expNum {
				return Result{Pass: true, Message: fmt.Sprintf("json %s %f > %f", path, actualNum, expNum)}
			}
			return Result{Pass: false, Message: fmt.Sprintf("json %s %f <= %f", path, actualNum, expNum)}
		}
		if actualNum < expNum {
			return Result{Pass: true, Message: fmt.Sprintf("json %s %f < %f", path, actualNum, expNum)}
		}
		return Result{Pass: false, Message: fmt.Sprintf("json %s %f >= %f", path, actualNum, expNum)}
	}

	// equality/inequality
	actual := val.Value()
	if actual == nil {
		return Result{Pass: false, Message: fmt.Sprintf("json path %s not found", path)}
	}

	// type-aware comparison
	switch exp := expect.(type) {
	case bool:
		if val.Type != gjson.True && val.Type != gjson.False {
			return Result{Pass: false, Message: fmt.Sprintf("json %s not a bool", path)}
		}
		actBool := val.Bool()
		pass := (op == "==" && actBool == exp) || (op == "!=" && actBool != exp)
		if pass {
			return Result{Pass: true, Message: fmt.Sprintf("json %s bool comparison pass", path)}
		}
		return Result{Pass: false, Message: fmt.Sprintf("json %s bool comparison fail (expect %v, got %v)", path, exp, actBool)}
	case int:
		if val.Type != gjson.Number {
			return Result{Pass: false, Message: fmt.Sprintf("json %s not a number", path)}
		}
		return compareNumbers(op, path, val.Num(), float64(exp))
	case int64:
		if val.Type != gjson.Number {
			return Result{Pass: false, Message: fmt.Sprintf("json %s not a number", path)}
		}
		return compareNumbers(op, path, val.Num(), float64(exp))
	case float64:
		if val.Type != gjson.Number {
			return Result{Pass: false, Message: fmt.Sprintf("json %s not a number", path)}
		}
		return compareNumbers(op, path, val.Num(), exp)
	case string:
		actualStr := val.String()
		pass := (op == "==" && actualStr == exp) || (op == "!=" && actualStr != exp)
		if pass {
			return Result{Pass: true, Message: fmt.Sprintf("json %s string comparison pass", path)}
		}
		return Result{Pass: false, Message: fmt.Sprintf("json %s string comparison fail (expect %s, got %s)", path, exp, actualStr)}
	default:
		actualStr := fmt.Sprint(actual)
		expectStr := fmt.Sprint(expect)
		pass := (op == "==" && actualStr == expectStr) || (op == "!=" && actualStr != expectStr)
		if pass {
			return Result{Pass: true, Message: fmt.Sprintf("json %s stringified comparison pass", path)}
		}
		return Result{Pass: false, Message: fmt.Sprintf("json %s stringified comparison fail (expect %s, got %s)", path, expectStr, actualStr)}
	}
}

func compareNumbers(op, path string, actual, expect float64) Result {
	pass := false
	switch op {
	case "==":
		pass = actual == expect
	case "!=":
		pass = actual != expect
	}
	if pass {
		return Result{Pass: true, Message: fmt.Sprintf("json %s number comparison pass", path)}
	}
	return Result{Pass: false, Message: fmt.Sprintf("json %s number comparison fail (expect %v, got %v)", path, expect, actual)}
}

func toFloat(v interface{}) float64 {
	switch val := v.(type) {
	case int:
		return float64(val)
	case int64:
		return float64(val)
	case float64:
		return val
	case string:
		f, _ := strconv.ParseFloat(val, 64)
		return f
	default:
		f, _ := strconv.ParseFloat(fmt.Sprint(v), 64)
		return f
	}
}
