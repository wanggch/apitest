package runner

import "testing"

func TestNormalizeVarKey(t *testing.T) {
	cases := map[string]string{
		`"foo"`:  "foo",
		`'bar'`:  "bar",
		" baz ":  "baz",
		"plain":  "plain",
		`"mid`:   `"mid`,
		`mid"`:   `mid"`,
		`'inner`: `'inner`,
	}
	for in, expect := range cases {
		if got := normalizeVarKey(in); got != expect {
			t.Fatalf("normalize %q => %q, expect %q", in, got, expect)
		}
	}
}
