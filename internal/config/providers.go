// Copyright (c) 2024 Ishaan Kalra
// SPDX-License-Identifier: MIT

package config

import (
	"bufio"
	"fmt"
	"os/exec"
	"regexp"
	"strings"

	"github.com/ishaankalra/terraform-state-downgrade/internal/state"
)

// Provider represents a provider with its module context
type Provider struct {
	Name    string
	Version string
	Module  string
}

type moduleFrame struct {
	depth int
	name  string
}

var providerRegex = regexp.MustCompile(`provider\[([^\]]+)\](?:\s+(\S+))?`)
var moduleRegex = regexp.MustCompile(`module\.(\S+)`)

// GetResourceProviderMappingFromState builds resource-to-provider mapping
// by reading the state file and parsing terraform providers output
func GetResourceProviderMappingFromState(configDir string, stateData *state.State) (map[string]string, error) {
	// Get list of providers from terraform providers command
	providers, err := ParseTerraformProviders(configDir)
	if err != nil {
		return nil, fmt.Errorf("failed to parse terraform providers: %w", err)
	}

	// Build a map of provider address -> module path
	providerModuleMap := make(map[string]string)
	for _, p := range providers {
		providerModuleMap[p.Name] = p.Module
	}

	// Build resource → provider mapping from state
	mapping := make(map[string]string)

	for _, resource := range stateData.Resources {
		// Skip data sources
		if resource.Mode != "managed" {
			continue
		}

		// Build resource address
		resourceAddr := fmt.Sprintf("%s.%s", resource.Type, resource.Name)

		// Extract provider address from state's provider field
		// Format: provider["registry.terraform.io/hashicorp/random"]
		providerAddr := ExtractProviderAddress(resource.Provider)

		// Check if this provider exists in the configuration
		if _, exists := providerModuleMap[providerAddr]; exists {
			mapping[resourceAddr] = providerAddr
		}
	}

	return mapping, nil
}

// ParseTerraformProviders runs `terraform providers` and parses the tree output
func ParseTerraformProviders(configDir string) ([]Provider, error) {
	cmd := exec.Command("terraform", "providers")
	cmd.Dir = configDir

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("terraform providers command failed: %w", err)
	}

	return parseProvidersOutput(string(output)), nil
}

// parseProvidersOutput parses the tree-based output of terraform providers
// Only parses "Providers required by configuration:" section, ignores "Providers required by state:"
func parseProvidersOutput(input string) []Provider {
	var providers []Provider
	scanner := bufio.NewScanner(strings.NewReader(input))
	var moduleStack []moduleFrame
	inConfigSection := false

	for scanner.Scan() {
		line := scanner.Text()

		// Check if we're entering the configuration section
		if strings.Contains(line, "Providers required by configuration:") {
			inConfigSection = true
			continue
		}

		// Stop parsing if we hit the state section
		if strings.Contains(line, "Providers required by state:") {
			break
		}

		// Only parse lines if we're in the configuration section
		if !inConfigSection {
			continue
		}

		depth := indentDepth(line)

		// Pop module stack if we've exited a module
		for len(moduleStack) > 0 && moduleStack[len(moduleStack)-1].depth >= depth {
			moduleStack = moduleStack[:len(moduleStack)-1]
		}

		// Check if this line is a module
		if modMatch := moduleRegex.FindStringSubmatch(line); modMatch != nil {
			parentPath := ""
			if len(moduleStack) > 0 {
				parentPath = moduleStack[len(moduleStack)-1].name
			}
			fullPath := modMatch[1]
			if parentPath != "" {
				fullPath = parentPath + "." + modMatch[1]
			}
			moduleStack = append(moduleStack, moduleFrame{depth: depth, name: fullPath})
			continue
		}

		// Check if this line is a provider
		if provMatch := providerRegex.FindStringSubmatch(line); provMatch != nil {
			modulePath := ""
			if len(moduleStack) > 0 {
				modulePath = moduleStack[len(moduleStack)-1].name
			}
			providers = append(providers, Provider{
				Name:    provMatch[1],
				Version: provMatch[2],
				Module:  modulePath,
			})
		}
	}
	return providers
}

// indentDepth calculates the depth based on tree characters
func indentDepth(line string) int {
	depth := 0
	for _, ch := range line {
		if ch == ' ' || ch == '│' || ch == '├' || ch == '└' || ch == '─' || ch == '|' {
			depth++
		} else {
			break
		}
	}
	return depth
}

// ExtractProviderAddress extracts provider address from state provider field
// Input: "provider[\"registry.terraform.io/hashicorp/aws\"]"
// Output: "registry.terraform.io/hashicorp/aws"
func ExtractProviderAddress(providerField string) string {
	re := regexp.MustCompile(`provider\["(.+?)"\]`)
	matches := re.FindStringSubmatch(providerField)
	if len(matches) > 1 {
		return matches[1]
	}
	// Fallback: just remove "provider[" and "]"
	result := strings.TrimPrefix(providerField, "provider[\"")
	result = strings.TrimSuffix(result, "\"]")
	return result
}