package runner_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"apitest/internal/config"
	"apitest/internal/report"
	"apitest/internal/runner"
)

func TestIntegrationFailFastAndReport(t *testing.T) {
	srv := newTestServer()
	defer srv.Close()

	planPath := filepath.Join(t.TempDir(), "plan.yaml")
	planContent := `name: "Demo"
base_url: "` + srv.URL + `"
steps:
  - name: "login"
    request:
      method: "POST"
      url: "/api/login"
      headers:
        Content-Type: "application/json"
      body:
        json:
          username: "alice"
          password: "secret"
    extract:
      token:
        from: "json"
        path: "data.token"
      user_id:
        from: "json"
        path: "data.user.id"
    assert:
      - type: "status"
        op: "=="
        expect: 200
  - name: "profile"
    request:
      method: "GET"
      url: "/api/users/{{user_id}}"
      headers:
        Authorization: "Bearer {{token}}"
    assert:
      - type: "status"
        op: "=="
        expect: 200
      - type: "json"
        path: "data.id"
        op: "=="
        expect: "999" # intentionally wrong to fail
  - name: "should not run"
    request:
      method: "GET"
      url: "/stop"
`
	if err := os.WriteFile(planPath, []byte(planContent), 0o644); err != nil {
		t.Fatalf("write plan: %v", err)
	}

	plan, err := config.LoadPlan(planPath)
	if err != nil {
		t.Fatalf("load plan: %v", err)
	}
	if plan.BaseURL == "" {
		t.Fatalf("base_url not parsed")
	}

	res := runner.Execute(plan, runner.RunnerOptions{Verbose: true})
	if res.Success {
		t.Fatalf("expected failure")
	}
	if res.FailedStep != "profile" {
		var errMsg string
		for _, s := range res.Steps {
			if !s.Success {
				errMsg = s.Error
				break
			}
		}
		t.Fatalf("unexpected failed step %s (%s)", res.FailedStep, errMsg)
	}
	if len(res.Steps) != 2 {
		t.Fatalf("expected 2 steps executed, got %d", len(res.Steps))
	}

	reportPath := filepath.Join(t.TempDir(), "report.md")
	if err := report.GenerateMarkdown(res, reportPath); err != nil {
		t.Fatalf("report: %v", err)
	}
	data, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	if !strings.Contains(string(data), "profile") {
		t.Fatalf("report missing step name")
	}
	if !strings.Contains(string(data), "FAIL") {
		t.Fatalf("report missing failure")
	}
	if res.EndTime.Before(res.StartTime) || res.EndTime.Sub(res.StartTime) < time.Millisecond {
		t.Fatalf("unexpected timestamps")
	}
}

func newTestServer() *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/login", func(w http.ResponseWriter, r *http.Request) {
		_ = r.Body.Close()
		resp := map[string]interface{}{
			"data": map[string]interface{}{
				"token": "token-abc",
				"user": map[string]interface{}{
					"id": "42",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})
	mux.HandleFunc("/api/users/42", func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer") {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		resp := map[string]interface{}{
			"data": map[string]interface{}{
				"id": "42",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})
	mux.HandleFunc("/stop", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	})
	return httptest.NewServer(mux)
}
