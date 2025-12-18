package runner

import (
	"testing"

	"apitest/internal/config"
	"apitest/internal/httpx"
)

func TestRunExtract(t *testing.T) {
	resp := httpx.ResponseInfo{
		Headers: map[string][]string{"X-Token": {"abc"}},
		Body:    `{"data": {"id": 5, "token": "def"}}`}

	out := map[string]string{}
	defs := map[string]config.ExtractDefinition{
		"token": {From: "json", Path: "data.token"},
		"id":    {From: "json", Path: "data.id"},
		"h":     {From: "header", Path: "X-Token"},
	}
	if err := runExtract(defs, resp, out); err != nil {
		t.Fatalf("extract: %v", err)
	}
	if out["token"] != "def" || out["id"] != "5" || out["h"] != "abc" {
		t.Fatalf("unexpected extract %#v", out)
	}
}

func TestRunExtractRegex(t *testing.T) {
	resp := httpx.ResponseInfo{Body: "order=12345, status=ok"}
	out := map[string]string{}
	defs := map[string]config.ExtractDefinition{
		"order": {From: "regex", Path: `order=(\d+)`, Group: 1},
	}
	if err := runExtract(defs, resp, out); err != nil {
		t.Fatalf("extract regex: %v", err)
	}
	if out["order"] != "12345" {
		t.Fatalf("unexpected regex %s", out["order"])
	}
}
