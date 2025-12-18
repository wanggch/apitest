package config

import (
	"os"
	"testing"
)

func TestLoadPlanBaseURL(t *testing.T) {
	data := []byte("name: test\nbase_url: https://example.com\nsteps:\n  - name: s1\n    request:\n      url: /\n")
	path := t.TempDir() + "/plan.yaml"
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	p, err := LoadPlan(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if p.BaseURL != "https://example.com" {
		t.Fatalf("unexpected base url %s", p.BaseURL)
	}
}
