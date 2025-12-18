package httpx

import (
	"context"
	"strings"
	"testing"
	"time"

	"apitest/internal/config"
)

func TestDoRequestRecordsInfoOnTemplateError(t *testing.T) {
	req := config.Request{
		Method: "POST",
		URL:    "/api/login",
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body: &config.RequestBody{
			JSON: map[string]interface{}{
				"username": "alice",
				"phone":    "{{phone}}",
				"password": "{{missing}}",
			},
		},
	}

	ri, _, err := DoRequest(
		context.Background(),
		BuildClient(time.Second, false),
		"https://example.com",
		req,
		map[string]string{"phone": "123"},
	)
	if err == nil {
		t.Fatalf("expected template error")
	}
	if !strings.Contains(err.Error(), "missing variable") {
		t.Fatalf("unexpected error: %v", err)
	}
	if ri.Method != "POST" {
		t.Fatalf("method not recorded, got %q", ri.Method)
	}
	if ri.URL != "https://example.com/api/login" {
		t.Fatalf("url not recorded, got %q", ri.URL)
	}
	if !strings.Contains(ri.Body, "alice") || !strings.Contains(ri.Body, "123") {
		t.Fatalf("body should include applied values, got: %s", ri.Body)
	}
	if ri.Headers["Content-Type"] == "" {
		t.Fatalf("headers not recorded")
	}
}
