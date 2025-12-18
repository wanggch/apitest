package runner

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/tidwall/gjson"

	"apitest/internal/assert"
	"apitest/internal/config"
	"apitest/internal/httpx"
	"apitest/internal/templ"
)

// RunnerOptions customize plan execution.
type RunnerOptions struct {
	BaseURL  string
	Vars     map[string]string
	Insecure bool
	Verbose  bool
}

// Result captures execution outcome.
type Result struct {
	PlanName   string
	Steps      []StepResult
	Success    bool
	FailedStep string
	StartTime  time.Time
	EndTime    time.Time
}

// StepResult describes per-step outcome.
type StepResult struct {
	Name       string
	Success    bool
	Error      string
	Request    httpx.RequestInfo
	Response   httpx.ResponseInfo
	Assertions []assert.Result
	Extracted  map[string]string
	StartTime  time.Time
	EndTime    time.Time
	VarError   string
}

// Execute runs the plan with provided options.
func Execute(plan *config.Plan, opts RunnerOptions) Result {
	res := Result{PlanName: plan.Name, StartTime: time.Now(), Success: true}
	baseURL := plan.BaseURL
	if opts.BaseURL != "" {
		baseURL = opts.BaseURL
	}

	ctx := make(map[string]string)
	for k, v := range plan.Vars {
		ctx[normalizeVarKey(k)] = fmt.Sprint(v)
	}
	ctx = templ.MergeContexts(ctx, opts.Vars)

	for _, step := range plan.Steps {
		sr := StepResult{Name: step.Name, StartTime: time.Now(), Success: true, Extracted: map[string]string{}}
		if opts.Verbose {
			fmt.Printf("==> Step: %s\n", step.Name)
		}
		client := httpx.BuildClient(time.Duration(step.Request.TimeoutMS)*time.Millisecond, opts.Insecure)
		reqInfo, respInfo, err := httpx.DoRequest(context.Background(), client, baseURL, step.Request, ctx)
		sr.Request = reqInfo
		sr.Response = respInfo
		sr.EndTime = time.Now()
		if err != nil {
			sr.Success = false
			sr.Error = err.Error()
			res.Steps = append(res.Steps, sr)
			res.Success = false
			res.FailedStep = step.Name
			break
		}

		// assertions
		sr.Assertions = assert.Evaluate(step.Assert, respInfo.Body, respInfo.Headers, respInfo.StatusCode, ctx)
		for _, ares := range sr.Assertions {
			if !ares.Pass {
				sr.Success = false
				sr.Error = ares.Message
				break
			}
		}

		// extraction executed even if no assertions? yes after assertions? maybe after? instructions? Use response; but if assertion fail should still record? stop, but we can skip extraction? maybe even after? We'll extract only if still success.
		if sr.Success {
			if err := runExtract(step.Extract, respInfo, sr.Extracted); err != nil {
				sr.Success = false
				sr.Error = err.Error()
			}
		}

		// merge extracted into ctx
		if sr.Success {
			for k, v := range sr.Extracted {
				ctx[k] = v
			}
		}

		if !sr.Success {
			res.Success = false
			res.FailedStep = step.Name
			res.Steps = append(res.Steps, sr)
			if opts.Verbose {
				fmt.Printf("Step %s failed: %s\n", step.Name, sr.Error)
			}
			break
		}

		res.Steps = append(res.Steps, sr)
		if opts.Verbose {
			fmt.Printf("Step %s passed\n", step.Name)
		}
	}

	res.EndTime = time.Now()
	return res
}

func normalizeVarKey(key string) string {
	key = strings.TrimSpace(key)
	if len(key) >= 2 {
		if (strings.HasPrefix(key, "\"") && strings.HasSuffix(key, "\"")) ||
			(strings.HasPrefix(key, "'") && strings.HasSuffix(key, "'")) {
			return key[1 : len(key)-1]
		}
	}
	return key
}

func runExtract(defs map[string]config.ExtractDefinition, resp httpx.ResponseInfo, out map[string]string) error {
	if len(defs) == 0 {
		return nil
	}
	for name, def := range defs {
		value, err := extractValue(def, resp)
		if err != nil {
			return fmt.Errorf("extract %s: %w", name, err)
		}
		out[name] = value
	}
	return nil
}

func extractValue(def config.ExtractDefinition, resp httpx.ResponseInfo) (string, error) {
	switch strings.ToLower(def.From) {
	case "json":
		if !gjson.Valid(resp.Body) {
			return "", fmt.Errorf("response not valid json")
		}
		val := gjson.Get(resp.Body, def.Path)
		if !val.Exists() {
			return "", fmt.Errorf("json path %s not found", def.Path)
		}
		return val.String(), nil
	case "header":
		vals := resp.Headers.Values(def.Path)
		if len(vals) == 0 {
			return "", fmt.Errorf("header %s not found", def.Path)
		}
		return vals[0], nil
	case "regex":
		pattern := def.Path
		group := def.Group
		if group == 0 {
			group = 1
		}
		re, err := regexp.Compile(pattern)
		if err != nil {
			return "", fmt.Errorf("invalid regex: %v", err)
		}
		matches := re.FindStringSubmatch(resp.Body)
		if len(matches) <= group {
			return "", fmt.Errorf("regex no group %d match", group)
		}
		return matches[group], nil
	default:
		return "", fmt.Errorf("unknown extract from %s", def.From)
	}
}
