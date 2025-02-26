package sdkdocs

import "github.com/nobl9/nobl9-go/manifest"

type ObjectDoc struct {
	Kind       manifest.Kind    `yaml:"kind"`
	Version    manifest.Version `yaml:"version"`
	Properties []PropertyDoc    `yaml:"properties"`
	Examples   []string         `yaml:"examples,omitempty"`
}

type PropertyDoc struct {
	Path          string     `json:"path"`
	Type          string     `json:"type"`
	Package       string     `json:"package,omitempty"`
	Doc           string     `yaml:"doc,omitempty"`
	IsDeprecated  bool       `json:"isDeprecated,omitempty"`
	IsOptional    bool       `json:"isOptional,omitempty"`
	IsSecret      bool       `json:"isSecret,omitempty"`
	Examples      []string   `json:"examples,omitempty"`
	Values        []string   `json:"values,omitempty"`
	Rules         []RulePlan `json:"rules,omitempty"`
	ChildrenPaths []string   `json:"childrenPaths,omitempty"`
}

type RulePlan struct {
	Description string   `json:"description"`
	Details     string   `json:"details,omitempty"`
	ErrorCode   string   `json:"errorCode,omitempty"`
	Conditions  []string `json:"conditions,omitempty"`
}
