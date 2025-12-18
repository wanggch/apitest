package assert

import (
	"net/http"
	"testing"

	"apitest/internal/config"
)

func TestJSONAssertions(t *testing.T) {
	body := `{"num": 10, "flag": true, "name": "bob", "arr": [1,2], "nested": {"token": "abc"}}`

	// exists
	res := Evaluate([]config.Assertion{{Type: "json", Path: "num", Op: "exists"}}, body, http.Header{}, 200, nil)
	if len(res) == 0 || !res[0].Pass {
		t.Fatalf("expected exists pass")
	}

	// equals number
	res = Evaluate([]config.Assertion{{Type: "json", Path: "num", Op: "==", Expect: 10}}, body, http.Header{}, 200, nil)
	if !res[0].Pass {
		t.Fatalf("expected number equality pass: %v", res[0].Message)
	}

	// bool mismatch
	res = Evaluate([]config.Assertion{{Type: "json", Path: "flag", Op: "!=", Expect: false}}, body, http.Header{}, 200, nil)
	if !res[0].Pass {
		t.Fatalf("expected bool inequality pass: %v", res[0].Message)
	}

	// contains
	res = Evaluate([]config.Assertion{{Type: "json", Path: "name", Op: "contains", Expect: "bo"}}, body, http.Header{}, 200, nil)
	if !res[0].Pass {
		t.Fatalf("expected contains pass: %v", res[0].Message)
	}

	// template expect
	res = Evaluate([]config.Assertion{{Type: "json", Path: "name", Op: "==", Expect: "{{who}}"}}, body, http.Header{}, 200, map[string]string{"who": "bob"})
	if !res[0].Pass {
		t.Fatalf("expected template pass: %v", res[0].Message)
	}

	// gt
	res = Evaluate([]config.Assertion{{Type: "json", Path: "num", Op: "gt", Expect: 5}}, body, http.Header{}, 200, nil)
	if !res[0].Pass {
		t.Fatalf("expected gt pass: %v", res[0].Message)
	}
}
