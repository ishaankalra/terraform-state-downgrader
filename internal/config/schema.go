// Copyright (c) 2024 Ishaan Kalra
// SPDX-License-Identifier: MIT

package config

import (
	"encoding/json"
	"fmt"
	"os/exec"
)

// GetSchemaVersions runs terraform providers schema -json and returns both version map and full schemas
// Returns:
//   - version map: provider address → resource type → schema version
//   - full schemas: provider address → resource type → full schema definition (with Block structure)
func GetSchemaVersions(configDir string) (map[string]map[string]int64, map[string]map[string]SchemaDefinition, error) {
	// Run terraform providers schema -json
	cmd := exec.Command("terraform", "providers", "schema", "-json")
	cmd.Dir = configDir

	output, err := cmd.Output()
	if err != nil {
		return nil, nil, fmt.Errorf("terraform providers schema failed: %w", err)
	}

	// Parse JSON
	var schemaOutput ProvidersSchemaOutput
	if err := json.Unmarshal(output, &schemaOutput); err != nil {
		return nil, nil, fmt.Errorf("failed to parse schema output: %w", err)
	}

	// Build provider → resource type → schema version map
	schemaVersions := make(map[string]map[string]int64)
	// Build provider → resource type → full schema definition map
	fullSchemas := make(map[string]map[string]SchemaDefinition)

	for providerAddr, providerSchema := range schemaOutput.ProviderSchemas {
		schemaVersions[providerAddr] = make(map[string]int64)
		fullSchemas[providerAddr] = make(map[string]SchemaDefinition)

		// Add resource schemas
		for resourceType, resourceSchema := range providerSchema.ResourceSchemas {
			schemaVersions[providerAddr][resourceType] = resourceSchema.Version
			fullSchemas[providerAddr][resourceType] = resourceSchema
		}

		// Add data source schemas (they also have schema versions)
		for dataSourceType, dataSourceSchema := range providerSchema.DataSourceSchemas {
			schemaVersions[providerAddr][dataSourceType] = dataSourceSchema.Version
			fullSchemas[providerAddr][dataSourceType] = dataSourceSchema
		}
	}

	return schemaVersions, fullSchemas, nil
}
