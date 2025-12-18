package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

const DefaultTimeoutMS = 10000

// Plan describes entire test plan.
type Plan struct {
	Name    string                 `yaml:"name" json:"name"`
	BaseURL string                 `yaml:"base_url" json:"base_url"`
	Vars    map[string]interface{} `yaml:"vars" json:"vars"`
	Steps   []Step                 `yaml:"steps" json:"steps"`
}

// Step describes a single request/assert sequence.
type Step struct {
	Name    string                       `yaml:"name" json:"name"`
	Request Request                      `yaml:"request" json:"request"`
	Extract map[string]ExtractDefinition `yaml:"extract" json:"extract"`
	Assert  []Assertion                  `yaml:"assert" json:"assert"`
}

// Request describes HTTP request properties.
type Request struct {
	Method    string                 `yaml:"method" json:"method"`
	URL       string                 `yaml:"url" json:"url"`
	Headers   map[string]string      `yaml:"headers" json:"headers"`
	Query     map[string]interface{} `yaml:"query" json:"query"`
	Body      *RequestBody           `yaml:"body" json:"body"`
	TimeoutMS int                    `yaml:"timeout_ms" json:"timeout_ms"`
}

// RequestBody holds mutually exclusive body encodings.
type RequestBody struct {
	Raw  string                 `yaml:"raw" json:"raw"`
	JSON interface{}            `yaml:"json" json:"json"`
	Form map[string]interface{} `yaml:"form" json:"form"`
}

// ExtractDefinition describes how to extract variables from response.
type ExtractDefinition struct {
	From  string `yaml:"from" json:"from"`
	Path  string `yaml:"path" json:"path"`
	Group int    `yaml:"group" json:"group"`
}

// Assertion describes supported assertion types.
type Assertion struct {
	Type   string      `yaml:"type" json:"type"`
	Op     string      `yaml:"op" json:"op"`
	Expect interface{} `yaml:"expect" json:"expect"`
	Name   string      `yaml:"name" json:"name"`
	Path   string      `yaml:"path" json:"path"`
}

// LoadPlan loads a YAML plan from file path.
func LoadPlan(path string) (*Plan, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read plan: %w", err)
	}
	var p Plan
	if err := yaml.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("parse plan: %w", err)
	}
	if len(p.Steps) == 0 {
		return nil, fmt.Errorf("plan has no steps")
	}
	for i := range p.Steps {
		if p.Steps[i].Request.TimeoutMS == 0 {
			p.Steps[i].Request.TimeoutMS = DefaultTimeoutMS
		}
	}
	return &p, nil
}
