// Copyright (c) 2024 Ishaan Kalra
// SPDX-License-Identifier: MIT

package analysis

import (
	"fmt"

	"github.com/ishaankalra/terraform-state-downgrade/internal/state"
)

// DetectMismatches cross-references state, configuration, and schema versions
// to find resources that need schema version downgrade
func DetectMismatches(
	stateData *state.State,
	resourceMapping map[string]string,
	schemaVersions map[string]map[string]int64,
) ([]Mismatch, error) {
	var mismatches []Mismatch

	// Process each resource in state
	for _, resource := range stateData.Resources {
		// Skip data sources, only process managed resources
		if resource.Mode != "managed" {
			continue
		}

		// Build resource address
		resourceAddr := fmt.Sprintf("%s.%s", resource.Type, resource.Name)

		// Get provider from configuration mapping
		providerAddr, ok := resourceMapping[resourceAddr]
		if !ok {
			// Resource not found in configuration - might have been removed
			// Skip it for now
			continue
		}

		// Get target schema version from provider
		providerSchemas, ok := schemaVersions[providerAddr]
		if !ok {
			// Provider not found in schema output
			continue
		}

		targetVersion, ok := providerSchemas[resource.Type]
		if !ok {
			// Resource type not found in provider schema
			continue
		}

		// Check each instance of the resource
		for instanceIdx, instance := range resource.Instances {
			currentVersion := int64(instance.SchemaVersion)

			// Check if downgrade is needed
			if currentVersion != targetVersion {
				// Extract resource ID
				resourceID := ""
				if id, ok := instance.Attributes["id"]; ok {
					if idStr, ok := id.(string); ok {
						resourceID = idStr
					}
				}

				// Extract timeouts
				timeouts := extractTimeouts(instance.Attributes)

				mismatch := Mismatch{
					ResourceAddress: resourceAddr,
					ResourceType:    resource.Type,
					ResourceName:    resource.Name,
					ProviderAddress: providerAddr,
					StateVersion:    currentVersion,
					TargetVersion:   targetVersion,
					ResourceID:      resourceID,
					Timeouts:        timeouts,
					InstanceIndex:   instanceIdx,
				}

				mismatches = append(mismatches, mismatch)
			}
		}
	}

	return mismatches, nil
}

// extractTimeouts extracts timeout configuration from resource attributes
func extractTimeouts(attributes map[string]interface{}) map[string]interface{} {
	if timeouts, ok := attributes["timeouts"]; ok {
		if timeoutsMap, ok := timeouts.(map[string]interface{}); ok {
			return timeoutsMap
		}
	}
	return nil
}