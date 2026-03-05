// Copyright (c) 2024 Ishaan Kalra
// SPDX-License-Identifier: MIT

package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
)

// ModulesManifest represents the .terraform/modules/modules.json file
type ModulesManifest struct {
	Modules []Module `json:"Modules"`
}

// Module represents a single module entry
type Module struct {
	Key    string `json:"Key"`
	Source string `json:"Source"`
	Dir    string `json:"Dir"`
}

// RequiredProvider represents a provider in required_providers block
type RequiredProvider struct {
	Name    string // Short name (e.g., "random")
	Source  string // Source (e.g., "hashicorp/random")
	Version string // Version constraint
	Module  string // Module key (e.g., "database", "network.subnet")
}

// ParseModulesManifest reads and parses .terraform/modules/modules.json
func ParseModulesManifest(configDir string) (*ModulesManifest, error) {
	manifestPath := filepath.Join(configDir, ".terraform", "modules", "modules.json")

	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read modules.json: %w", err)
	}

	var manifest ModulesManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse modules.json: %w", err)
	}

	return &manifest, nil
}

// ParseRequiredProviders extracts required_providers from all .tf files in a directory
func ParseRequiredProviders(configDir, moduleDir, moduleKey string) ([]RequiredProvider, error) {
	fullPath := filepath.Join(configDir, moduleDir)

	// Find all .tf files
	tfFiles, err := filepath.Glob(filepath.Join(fullPath, "*.tf"))
	if err != nil {
		return nil, fmt.Errorf("failed to glob .tf files: %w", err)
	}

	var providers []RequiredProvider
	parser := hclparse.NewParser()

	for _, tfFile := range tfFiles {
		src, err := os.ReadFile(tfFile)
		if err != nil {
			continue // Skip files we can't read
		}

		file, diags := parser.ParseHCL(src, tfFile)
		if diags.HasErrors() {
			continue // Skip files with parse errors
		}

		// Extract required_providers
		extracted := extractRequiredProvidersFromFile(file.Body, moduleKey)
		providers = append(providers, extracted...)
	}

	return providers, nil
}

// extractRequiredProvidersFromFile parses the HCL body for required_providers blocks
func extractRequiredProvidersFromFile(body hcl.Body, moduleKey string) []RequiredProvider {
	var providers []RequiredProvider

	content, _, diags := body.PartialContent(&hcl.BodySchema{
		Blocks: []hcl.BlockHeaderSchema{
			{Type: "terraform"},
		},
	})

	if diags.HasErrors() {
		return providers
	}

	for _, block := range content.Blocks {
		if block.Type != "terraform" {
			continue
		}

		// Look for required_providers block inside terraform block
		tfContent, _, diags := block.Body.PartialContent(&hcl.BodySchema{
			Blocks: []hcl.BlockHeaderSchema{
				{Type: "required_providers"},
			},
		})

		if diags.HasErrors() {
			continue
		}

		for _, rpBlock := range tfContent.Blocks {
			if rpBlock.Type != "required_providers" {
				continue
			}

			// Parse provider configurations
			attrs, diags := rpBlock.Body.JustAttributes()
			if diags.HasErrors() {
				continue
			}

			for name, attr := range attrs {
				provider := RequiredProvider{
					Name:   name,
					Module: moduleKey,
				}

				// Parse the provider configuration
				val, diags := attr.Expr.Value(nil)
				if diags.HasErrors() {
					continue
				}

				if val.Type().IsObjectType() {
					// Extract source and version
					if sourceVal := val.GetAttr("source"); !sourceVal.IsNull() {
						provider.Source = sourceVal.AsString()
					}
					if versionVal := val.GetAttr("version"); !versionVal.IsNull() {
						provider.Version = versionVal.AsString()
					}
				}

				providers = append(providers, provider)
			}
		}
	}

	return providers
}

// GetAllRequiredProviders gets required_providers from all modules
func GetAllRequiredProviders(configDir string) ([]RequiredProvider, error) {
	var allProviders []RequiredProvider

	// Always parse the root directory first
	rootProviders, err := ParseRequiredProviders(configDir, ".", "")
	if err != nil {
		// Root directory should be parseable, but don't fail completely
		// Some projects might not have .tf files in root
	} else {
		allProviders = append(allProviders, rootProviders...)
	}

	// Try to parse modules manifest (may not exist if no modules)
	manifest, err := ParseModulesManifest(configDir)
	if err != nil {
		// No modules.json is fine - project might not have modules
		// Just return the root providers
		return allProviders, nil
	}

	// Parse required_providers for each module
	for _, module := range manifest.Modules {
		// Skip root module (already parsed above)
		if module.Key == "" || module.Dir == "." {
			continue
		}

		providers, err := ParseRequiredProviders(configDir, module.Dir, module.Key)
		if err != nil {
			// Log but don't fail - some modules might not have providers
			continue
		}
		allProviders = append(allProviders, providers...)
	}

	return allProviders, nil
}

// MapProvidersToFQDN maps provider sources to FQDNs using the lock file
func MapProvidersToFQDN(providers []RequiredProvider, lockFile *LockFile) map[string]string {
	// Build a map of source → FQDN from lock file
	sourceFQDNMap := make(map[string]string)

	for fqdn := range lockFile.Providers {
		// Extract source from FQDN
		// "registry.terraform.io/hashicorp/random" → "hashicorp/random"
		parts := strings.Split(fqdn, "/")
		if len(parts) >= 3 {
			source := strings.Join(parts[len(parts)-2:], "/")
			sourceFQDNMap[source] = fqdn
		}
	}

	// Build result map: provider short name → FQDN
	result := make(map[string]string)
	for _, p := range providers {
		if p.Source == "" {
			continue
		}

		if fqdn, exists := sourceFQDNMap[p.Source]; exists {
			result[p.Name] = fqdn
		}
	}

	return result
}

// BuildModuleProviderMapping builds a mapping of (module, provider_name) → FQDN
// This handles cases where different modules use different versions of the same provider
func BuildModuleProviderMapping(configDir string) (map[string]map[string]string, error) {
	// Get all required providers from all modules
	providers, err := GetAllRequiredProviders(configDir)
	if err != nil {
		return nil, fmt.Errorf("failed to get required providers: %w", err)
	}

	// Parse lock file to resolve sources to FQDNs
	lockFile, err := ParseLockFile(configDir)
	if err != nil {
		return nil, fmt.Errorf("failed to parse lock file: %w", err)
	}

	// Build a map of source → FQDN from lock file
	sourceFQDNMap := make(map[string]string)
	for fqdn := range lockFile.Providers {
		// Extract source from FQDN
		// "registry.terraform.io/hashicorp/aws3" → "hashicorp/aws3"
		parts := strings.Split(fqdn, "/")
		if len(parts) >= 3 {
			source := strings.Join(parts[len(parts)-2:], "/")
			sourceFQDNMap[source] = fqdn
		}
	}

	// Build result: module → (provider_name → FQDN)
	result := make(map[string]map[string]string)

	for _, p := range providers {
		if p.Source == "" {
			continue
		}

		// Get FQDN for this provider source
		fqdn, exists := sourceFQDNMap[p.Source]
		if !exists {
			continue
		}

		// Initialize map for this module if needed
		if result[p.Module] == nil {
			result[p.Module] = make(map[string]string)
		}

		// Map: module → provider_name → FQDN
		// e.g., "database" → "aws" → "registry.terraform.io/hashicorp/aws3"
		result[p.Module][p.Name] = fqdn
	}

	return result, nil
}
