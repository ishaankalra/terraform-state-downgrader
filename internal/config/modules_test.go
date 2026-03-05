// Copyright (c) 2024 Ishaan Kalra
// SPDX-License-Identifier: MIT

package config

import (
	"encoding/json"
	"path/filepath"
	"testing"
)

func TestGetAllRequiredProviders_WithModules(t *testing.T) {
	// Use the sample_with_modules test directory
	configDir := filepath.Join("..", "..", "tests", "sample_with_modules")

	providers, err := GetAllRequiredProviders(configDir)
	if err != nil {
		t.Fatalf("Failed to get required providers: %v", err)
	}

	if len(providers) == 0 {
		t.Fatal("Expected to find providers, got 0")
	}

	t.Logf("Found %d required providers", len(providers))

	// Pretty print the providers
	jsonData, _ := json.MarshalIndent(providers, "", "  ")
	t.Logf("Providers:\n%s", string(jsonData))

	// Verify we have providers from different modules
	moduleCount := make(map[string]int)
	for _, p := range providers {
		moduleCount[p.Module]++
	}

	t.Logf("Provider count by module:")
	for module, count := range moduleCount {
		if module == "" {
			t.Logf("  Root: %d providers", count)
		} else {
			t.Logf("  %s: %d providers", module, count)
		}
	}

	// Verify root module has providers
	if moduleCount[""] == 0 {
		t.Error("Expected providers in root module")
	}

	// Check for specific providers
	expectedProviders := map[string]string{
		"random": "hashicorp/random",
		"null":   "hashicorp/null",
		"local":  "hashicorp/local",
		"time":   "hashicorp/time",
	}

	found := make(map[string]bool)
	for _, p := range providers {
		if expectedSource, exists := expectedProviders[p.Name]; exists {
			if p.Source == expectedSource {
				found[p.Name] = true
			}
		}
	}

	for name, source := range expectedProviders {
		if !found[name] {
			t.Errorf("Expected to find provider %s with source %s", name, source)
		}
	}
}

func TestGetAllRequiredProviders_WithoutModules(t *testing.T) {
	// Use the sample_with_modules root directory but without modules.json
	// This tests parsing only root .tf files
	configDir := filepath.Join("..", "..", "tests", "sample_with_modules")

	// Just parse the root directory directly
	providers, err := ParseRequiredProviders(configDir, ".", "")
	if err != nil {
		t.Fatalf("Failed to get required providers: %v", err)
	}

	if len(providers) == 0 {
		t.Fatal("Expected to find providers in root, got 0")
	}

	t.Logf("Found %d required providers (root only, no modules)", len(providers))

	// All providers should be from root module
	for _, p := range providers {
		if p.Module != "" {
			t.Errorf("Expected all providers to be from root module, got module: %s", p.Module)
		}
	}

	// Should have random and null provider from root
	foundRandom := false
	foundNull := false
	for _, p := range providers {
		if p.Name == "random" {
			foundRandom = true
			if p.Source != "hashicorp/random" {
				t.Errorf("Expected random provider source to be hashicorp/random, got: %s", p.Source)
			}
		}
		if p.Name == "null" {
			foundNull = true
			if p.Source != "hashicorp/null" {
				t.Errorf("Expected null provider source to be hashicorp/null, got: %s", p.Source)
			}
		}
	}

	if !foundRandom {
		t.Error("Expected to find random provider")
	}
	if !foundNull {
		t.Error("Expected to find null provider")
	}
}

func TestMapProvidersToFQDN(t *testing.T) {
	configDir := filepath.Join("..", "..", "tests", "sample_with_modules")

	// Get required providers
	providers, err := GetAllRequiredProviders(configDir)
	if err != nil {
		t.Fatalf("Failed to get required providers: %v", err)
	}

	// Parse lock file
	lockFile, err := ParseLockFile(configDir)
	if err != nil {
		t.Fatalf("Failed to parse lock file: %v", err)
	}

	// Map to FQDNs
	fqdnMap := MapProvidersToFQDN(providers, lockFile)

	if len(fqdnMap) == 0 {
		t.Fatal("Expected to map providers to FQDNs, got 0")
	}

	t.Logf("Mapped %d providers to FQDNs", len(fqdnMap))

	// Pretty print the mapping
	jsonData, _ := json.MarshalIndent(fqdnMap, "", "  ")
	t.Logf("Provider name → FQDN mapping:\n%s", string(jsonData))

	// Verify specific mappings
	expectedMappings := map[string]string{
		"random": "registry.terraform.io/hashicorp/random",
		"null":   "registry.terraform.io/hashicorp/null",
		"local":  "registry.terraform.io/hashicorp/local",
		"time":   "registry.terraform.io/hashicorp/time",
	}

	for name, expectedFQDN := range expectedMappings {
		actualFQDN, exists := fqdnMap[name]
		if !exists {
			t.Errorf("Expected to find FQDN for provider %s", name)
			continue
		}
		if actualFQDN != expectedFQDN {
			t.Errorf("Provider %s: expected FQDN %s, got %s", name, expectedFQDN, actualFQDN)
		}
	}
}