package report

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"apitest/internal/runner"
)

const (
	maxReportBodyLength = 4000
)

// GenerateMarkdown builds a markdown report file for the run result.
func GenerateMarkdown(result runner.Result, outputPath string) error {
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return fmt.Errorf("create report dir: %w", err)
	}
	f, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("create report: %w", err)
	}
	defer f.Close()

	writeLine := func(s string) {
		_, _ = f.WriteString(s + "\n")
	}

	writeLine(fmt.Sprintf("# %s", result.PlanName))
	writeLine("")
	writeLine("## Summary")
	writeLine("")
	writeLine("| Item | Value |")
	writeLine("| --- | --- |")
	writeLine(fmt.Sprintf("| Start | %s |", result.StartTime.Format(time.RFC3339)))
	writeLine(fmt.Sprintf("| End | %s |", result.EndTime.Format(time.RFC3339)))
	writeLine(fmt.Sprintf("| Duration | %s |", result.EndTime.Sub(result.StartTime)))
	status := "PASS"
	failed := result.FailedStep
	if !result.Success {
		status = "FAIL"
		if failed == "" {
			failed = "unknown"
		}
	}
	writeLine(fmt.Sprintf("| Result | %s |", status))
	if failed != "" {
		writeLine(fmt.Sprintf("| Failed Step | %s |", failed))
	}
	writeLine("")

	for _, step := range result.Steps {
		writeStep(writeLine, step)
	}

	return nil
}

func writeStep(writeLine func(string), step runner.StepResult) {
	status := "PASS"
	if !step.Success {
		status = "FAIL"
	}
	writeLine(fmt.Sprintf("## Step: %s (%s)", step.Name, status))
	writeLine("")
	writeLine(fmt.Sprintf("- Duration: %s", step.EndTime.Sub(step.StartTime)))
	if step.Error != "" {
		writeLine(fmt.Sprintf("- Error: %s", step.Error))
	}
	writeLine("")

	writeLine("### Request")
	writeLine("")
	writeLine(fmt.Sprintf("- Method: %s", step.Request.Method))
	writeLine(fmt.Sprintf("- URL: %s", step.Request.URL))
	if len(step.Request.Query) > 0 {
		writeLine("- Query:")
		for k, v := range step.Request.Query {
			writeLine(fmt.Sprintf("  - %s: %s", k, maskSensitive(k, v)))
		}
	}
	if len(step.Request.Headers) > 0 {
		writeLine("- Headers:")
		for k, v := range step.Request.Headers {
			writeLine(fmt.Sprintf("  - %s: %s", k, maskSensitive(k, v)))
		}
	}
	if step.Request.Body != "" {
		writeLine("- Body:")
		writeLine("```")
		writeLine(truncateBody(formatBody(maskBody(step.Request.Body))))
		writeLine("```")
	}

	writeLine("")
	writeLine("### Response")
	writeLine("")
	writeLine(fmt.Sprintf("- Status: %d", step.Response.StatusCode))
	if len(step.Response.Headers) > 0 {
		writeLine("- Headers:")
		for k, v := range step.Response.Headers {
			writeLine(fmt.Sprintf("  - %s: %s", k, maskSensitive(k, strings.Join(v, ","))))
		}
	}
	if step.Response.Body != "" {
		bodyText := truncateBody(formatBody(maskBody(step.Response.Body)))
		if step.Response.BodyTruncated {
			bodyText += "\n... (truncated)"
		}
		writeLine("- Body:")
		writeLine("```")
		writeLine(bodyText)
		writeLine("```")
	}

	if len(step.Extracted) > 0 {
		writeLine("")
		writeLine("### Extracted Vars")
		for k, v := range step.Extracted {
			writeLine(fmt.Sprintf("- %s: %s", k, maskSensitive(k, v)))
		}
	}

	if len(step.Assertions) > 0 {
		writeLine("")
		writeLine("### Assertions")
		for i, ar := range step.Assertions {
			prefix := "FAIL"
			if ar.Pass {
				prefix = "PASS"
			}
			writeLine(fmt.Sprintf("%d. **%s** %s", i+1, prefix, ar.Message))
		}
	}

	writeLine("")
}

func truncateBody(body string) string {
	if len(body) <= maxReportBodyLength {
		return body
	}
	return body[:maxReportBodyLength] + "\n... (truncated)"
}

func maskSensitive(key, value string) string {
	lower := strings.ToLower(key)
	if lower == "authorization" || strings.Contains(lower, "token") || strings.Contains(lower, "password") || strings.Contains(lower, "secret") {
		return "[masked]"
	}
	return value
}

func maskBody(body string) string {
	replacer := body
	patterns := []string{"token", "password", "secret"}
	for _, p := range patterns {
		re := regexp.MustCompile(fmt.Sprintf(`(?i)("?%s"?\s*[:=]\s*)("?)([^"\n ]+)`, p))
		replacer = re.ReplaceAllString(replacer, "$1$2[masked]")
	}
	return replacer
}

// formatBody prettifies JSON bodies for readability; falls back to original text on failure.
func formatBody(body string) string {
	if body == "" {
		return body
	}
	var buf bytes.Buffer
	if err := json.Indent(&buf, []byte(body), "", "  "); err != nil {
		return body
	}
	return buf.String()
}
