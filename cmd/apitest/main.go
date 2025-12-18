package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"apitest/internal/config"
	"apitest/internal/report"
	"apitest/internal/runner"
	"apitest/internal/templ"
)

type stringList []string

func (s *stringList) String() string { return strings.Join(*s, ",") }
func (s *stringList) Set(v string) error {
	*s = append(*s, v)
	return nil
}

func main() {
	if len(os.Args) < 2 || os.Args[1] != "run" {
		fmt.Fprintf(os.Stderr, "Usage: %s run -f plan.yaml [options]\n", filepath.Base(os.Args[0]))
		os.Exit(2)
	}

	fs := flag.NewFlagSet("run", flag.ExitOnError)
	var planFile string
	var output string
	var baseURL string
	var insecure bool
	var verbose bool
	var envFile string
	var vars stringList

	fs.StringVar(&planFile, "f", "", "Path to plan YAML file")
	fs.StringVar(&planFile, "file", "", "Path to plan YAML file")
	fs.StringVar(&output, "o", "report.md", "Output report markdown path")
	fs.StringVar(&output, "output", "report.md", "Output report markdown path")
	fs.StringVar(&baseURL, "base-url", "", "Override base URL")
	fs.BoolVar(&insecure, "insecure", false, "Skip TLS verification")
	fs.BoolVar(&verbose, "verbose", false, "Verbose execution log")
	fs.StringVar(&envFile, "env", "", "Additional vars yaml file")
	fs.Var(&vars, "var", "Extra variable k=v (repeatable)")

	if err := fs.Parse(os.Args[2:]); err != nil {
		os.Exit(2)
	}

	if planFile == "" {
		fmt.Fprintln(os.Stderr, "-f / --file is required")
		os.Exit(2)
	}

	exitCode := execute(planFile, output, baseURL, insecure, verbose, envFile, vars)
	os.Exit(exitCode)
}

func execute(planFile, output, baseURL string, insecure, verbose bool, envFile string, vars []string) int {
	plan, err := config.LoadPlan(planFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load plan: %v\n", err)
		return 2
	}

	cliVars, err := parseVars(vars)
	if err != nil {
		fmt.Fprintf(os.Stderr, "vars: %v\n", err)
		return 2
	}

	envVars := map[string]string{}
	if envFile != "" {
		envVars, err = loadEnvFile(envFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "env file: %v\n", err)
			return 2
		}
	}

	allVars := templ.MergeContexts(envVars, cliVars)

	res := runner.Execute(plan, runner.RunnerOptions{
		BaseURL:  baseURL,
		Vars:     allVars,
		Insecure: insecure,
		Verbose:  verbose,
	})

	if err := ensureDir(output); err != nil {
		fmt.Fprintf(os.Stderr, "report path: %v\n", err)
		return 2
	}

	if err := report.GenerateMarkdown(res, output); err != nil {
		fmt.Fprintf(os.Stderr, "generate report: %v\n", err)
		return 1
	}

	if res.Success {
		return 0
	}
	return 1
}

func parseVars(items []string) (map[string]string, error) {
	out := make(map[string]string)
	for _, item := range items {
		parts := strings.SplitN(item, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid var %s, expect k=v", item)
		}
		out[parts[0]] = parts[1]
	}
	return out, nil
}

func loadEnvFile(path string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	raw := map[string]interface{}{}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	out := make(map[string]string)
	for k, v := range raw {
		out[k] = fmt.Sprint(v)
	}
	return out, nil
}

func ensureDir(path string) error {
	dir := filepath.Dir(path)
	if dir == "." || dir == "" {
		return nil
	}
	return os.MkdirAll(dir, 0o755)
}
