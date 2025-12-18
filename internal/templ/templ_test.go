package templ

import "testing"

func TestApplyStringMissingVar(t *testing.T) {
	_, err := ApplyString("hello {{name}}", map[string]string{"other": "x"})
	if err == nil {
		t.Fatalf("expected error for missing var")
	}
}

func TestApplyInterfaceNested(t *testing.T) {
	input := map[string]interface{}{
		"user": map[string]interface{}{
			"name":    "{{name}}",
			"age":     30,
			"hobbies": []interface{}{"{{hobby}}", "reading"},
		},
	}
	ctx := map[string]string{"name": "alice", "hobby": "coding"}
	outRaw, err := ApplyInterface(input, ctx)
	if err != nil {
		t.Fatalf("apply interface: %v", err)
	}
	out := outRaw.(map[string]interface{})
	user := out["user"].(map[string]interface{})
	if user["name"].(string) != "alice" {
		t.Fatalf("unexpected name %v", user["name"])
	}
	hobbies := user["hobbies"].([]interface{})
	if hobbies[0].(string) != "coding" {
		t.Fatalf("unexpected hobby %v", hobbies[0])
	}
	if hobbies[1].(string) != "reading" {
		t.Fatalf("unexpected hobby2 %v", hobbies[1])
	}
}
