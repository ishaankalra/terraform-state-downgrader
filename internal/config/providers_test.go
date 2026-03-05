// Copyright (c) 2024 Ishaan Kalra
// SPDX-License-Identifier: MIT

package config

import (
	"testing"
)

func TestParseProvidersOutput_Simple(t *testing.T) {
	input := `Providers required by configuration:
.
└── provider[registry.terraform.io/hashicorp/random] 3.5.1
`

	providers := parseProvidersOutput(input)

	if len(providers) != 1 {
		t.Fatalf("Expected 1 provider, got %d", len(providers))
	}

	p := providers[0]
	if p.Name != "registry.terraform.io/hashicorp/random" {
		t.Errorf("Expected name 'registry.terraform.io/hashicorp/random', got %s", p.Name)
	}
	if p.Version != "3.5.1" {
		t.Errorf("Expected version '3.5.1', got %s", p.Version)
	}
	if p.Module != "" {
		t.Errorf("Expected empty module (root), got %s", p.Module)
	}
}

func TestParseProvidersOutput_MultipleProviders(t *testing.T) {
	input := `Providers required by configuration:
.
├── provider[registry.terraform.io/hashicorp/random] 3.5.1
├── provider[registry.terraform.io/hashicorp/aws] 3.74.0
└── provider[registry.terraform.io/hashicorp/local] 2.2.3
`

	providers := parseProvidersOutput(input)

	if len(providers) != 3 {
		t.Fatalf("Expected 3 providers, got %d", len(providers))
	}

	expectedProviders := map[string]string{
		"registry.terraform.io/hashicorp/random": "3.5.1",
		"registry.terraform.io/hashicorp/aws":    "3.74.0",
		"registry.terraform.io/hashicorp/local":  "2.2.3",
	}

	for _, p := range providers {
		expectedVersion, exists := expectedProviders[p.Name]
		if !exists {
			t.Errorf("Unexpected provider: %s", p.Name)
			continue
		}
		if p.Version != expectedVersion {
			t.Errorf("Provider %s: expected version %s, got %s", p.Name, expectedVersion, p.Version)
		}
	}
}

func TestParseProvidersOutput_WithModule(t *testing.T) {
	input := `Providers required by configuration:
.
├── provider[registry.terraform.io/hashicorp/random] 3.5.1
└── module.database
    └── provider[registry.terraform.io/hashicorp/aws] 3.74.0
`

	providers := parseProvidersOutput(input)

	if len(providers) != 2 {
		t.Fatalf("Expected 2 providers, got %d", len(providers))
	}

	// Check root provider
	var rootProvider *Provider
	var moduleProvider *Provider
	for i := range providers {
		if providers[i].Module == "" {
			rootProvider = &providers[i]
		} else {
			moduleProvider = &providers[i]
		}
	}

	if rootProvider == nil {
		t.Fatal("Root provider not found")
	}
	if rootProvider.Name != "registry.terraform.io/hashicorp/random" {
		t.Errorf("Expected random provider at root, got %s", rootProvider.Name)
	}

	if moduleProvider == nil {
		t.Fatal("Module provider not found")
	}
	if moduleProvider.Module != "database" {
		t.Errorf("Expected module 'database', got %s", moduleProvider.Module)
	}
	if moduleProvider.Name != "registry.terraform.io/hashicorp/aws" {
		t.Errorf("Expected aws provider in module, got %s", moduleProvider.Name)
	}
}

func TestParseProvidersOutput_NestedModules(t *testing.T) {
	input := `Providers required by configuration:
.
├── provider[registry.terraform.io/hashicorp/random] 3.5.1
├── module.level2
│   ├── provider[registry.terraform.io/hashicorp/helm] 2.8.0
│   └── module.mongo
│       ├── provider[registry.terraform.io/hashicorp/aws3]
│       └── module.mongo-pvc
│           └── provider[registry.terraform.io/hashicorp/helm]
`

	providers := parseProvidersOutput(input)

	if len(providers) != 4 {
		t.Fatalf("Expected 4 providers, got %d", len(providers))
	}

	// Check module paths
	moduleMap := make(map[string]string)
	for _, p := range providers {
		moduleMap[p.Name] = p.Module
	}

	// Root level
	if moduleMap["registry.terraform.io/hashicorp/random"] != "" {
		t.Errorf("random should be at root, got module: %s", moduleMap["registry.terraform.io/hashicorp/random"])
	}

	// First level module
	helmModule := moduleMap["registry.terraform.io/hashicorp/helm"]
	if helmModule != "level2" && helmModule != "level2.mongo" && helmModule != "level2.mongo.mongo-pvc" {
		t.Errorf("helm should be in a level2 module, got: %s", helmModule)
	}

	// Nested module
	if moduleMap["registry.terraform.io/hashicorp/aws3"] != "level2.mongo" {
		t.Errorf("aws3 should be in level2.mongo, got: %s", moduleMap["registry.terraform.io/hashicorp/aws3"])
	}
}

func TestParseProvidersOutput_NoVersion(t *testing.T) {
	input := `Providers required by configuration:
.
└── provider[registry.terraform.io/facets-cloud/facets]
`

	providers := parseProvidersOutput(input)

	if len(providers) != 1 {
		t.Fatalf("Expected 1 provider, got %d", len(providers))
	}

	p := providers[0]
	if p.Name != "registry.terraform.io/facets-cloud/facets" {
		t.Errorf("Expected facets provider, got %s", p.Name)
	}
	if p.Version != "" {
		t.Errorf("Expected empty version, got %s", p.Version)
	}
}

func TestParseProvidersOutput_IgnoresStateSection(t *testing.T) {
	input := `Providers required by configuration:
.
├── provider[registry.terraform.io/hashicorp/random] 3.5.1
└── provider[registry.terraform.io/hashicorp/aws] 3.74.0

Providers required by state:

provider[registry.terraform.io/hashicorp/external]
provider[registry.terraform.io/hashicorp/null]
provider[registry.terraform.io/hashicorp/template]
`

	providers := parseProvidersOutput(input)

	// Should only parse configuration section (2 providers), not state section (3 providers)
	if len(providers) != 2 {
		t.Fatalf("Expected 2 providers from config section, got %d", len(providers))
	}

	// Verify we got the right providers from config section
	foundRandom := false
	foundAWS := false
	for _, p := range providers {
		if p.Name == "registry.terraform.io/hashicorp/random" {
			foundRandom = true
		}
		if p.Name == "registry.terraform.io/hashicorp/aws" {
			foundAWS = true
		}
		// Make sure we didn't pick up providers from state section
		if p.Name == "registry.terraform.io/hashicorp/external" ||
			p.Name == "registry.terraform.io/hashicorp/null" ||
			p.Name == "registry.terraform.io/hashicorp/template" {
			t.Errorf("Should not have parsed provider from state section: %s", p.Name)
		}
	}

	if !foundRandom || !foundAWS {
		t.Error("Expected to find random and aws providers from config section")
	}
}

func TestIndentDepth(t *testing.T) {
	tests := []struct {
		line     string
		expected int
	}{
		{"provider[foo]", 0},
		{"├── provider[foo]", 4},
		{"│   ├── provider[foo]", 8},
		{"│   │   └── provider[foo]", 12},
		{"    provider[foo]", 4},
	}

	for _, tt := range tests {
		depth := indentDepth(tt.line)
		if depth != tt.expected {
			t.Errorf("indentDepth(%q) = %d, expected %d", tt.line, depth, tt.expected)
		}
	}
}
