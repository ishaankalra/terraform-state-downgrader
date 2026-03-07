// Copyright (c) 2024 Ishaan Kalra
// SPDX-License-Identifier: MIT

package config

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/ishaankalra/terraform-state-downgrader/internal/state"
)

// GetResourceProviderMappingFromState builds resource-to-provider mapping
// by extracting provider information directly from state
// Resource addresses include module path and indices for count/for_each
// (e.g., "module.database.aws_s3_bucket.example[0]")
func GetResourceProviderMappingFromState(configDir string, stateData *state.State) (map[string]string, error) {
	// Build resource → provider mapping from state
	mapping := make(map[string]string)

	for _, resource := range stateData.Resources {
		// Skip data sources
		if resource.Mode != "managed" {
			continue
		}

		// Build base resource address including module path
		var baseAddr string
		if resource.Module != "" {
			// Module resource: "module.database.aws_s3_bucket.example"
			baseAddr = fmt.Sprintf("%s.%s.%s", resource.Module, resource.Type, resource.Name)
		} else {
			// Root resource: "aws_s3_bucket.example"
			baseAddr = fmt.Sprintf("%s.%s", resource.Type, resource.Name)
		}

		// Extract provider address from state's provider field
		// Format: provider["registry.terraform.io/hashicorp/random"]
		// Output: "registry.terraform.io/hashicorp/random"
		providerAddr := ExtractProviderAddress(resource.Provider)

		if providerAddr == "" {
			continue
		}

		// Map each instance (handles count/for_each)
		for _, instance := range resource.Instances {
			instanceAddr := baseAddr
			if instance.IndexKey != nil {
				switch key := instance.IndexKey.(type) {
				case float64:
					// count: resource.name[0]
					instanceAddr = fmt.Sprintf("%s[%d]", baseAddr, int(key))
				case string:
					// for_each: resource.name["key"]
					instanceAddr = fmt.Sprintf("%s[\"%s\"]", baseAddr, key)
				}
			}
			mapping[instanceAddr] = providerAddr
		}
	}

	return mapping, nil
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