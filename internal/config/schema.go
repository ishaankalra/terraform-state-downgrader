// Copyright (c) 2024 Ishaan Kalra
// SPDX-License-Identifier: MIT

package config

import (
	"encoding/json"
	"fmt"
	"os/exec"
)

// GetSchemaVersions runs terraform providers schema -json and extracts schema versions
// Returns: provider address → resource type → schema version
func GetSchemaVersions(configDir string) (map[string]map[string]int64, error) {
	// Run terraform providers schema -json
	cmd := exec.Command("terraform", "providers", "schema", "-json")
	cmd.Dir = configDir

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("terraform providers schema failed: %w", err)
	}

	// Parse JSON
	var schemaOutput ProvidersSchemaOutput
	if err := json.Unmarshal(output, &schemaOutput); err != nil {
		return nil, fmt.Errorf("failed to parse schema output: %w", err)
	}

	// Build provider → resource type → schema version map
	schemaVersions := make(map[string]map[string]int64)

	for providerAddr, providerSchema := range schemaOutput.ProviderSchemas {
		schemaVersions[providerAddr] = make(map[string]int64)

		// Add resource schemas
		for resourceType, resourceSchema := range providerSchema.ResourceSchemas {
			schemaVersions[providerAddr][resourceType] = resourceSchema.Version
		}

		// Add data source schemas (they also have schema versions)
		for dataSourceType, dataSourceSchema := range providerSchema.DataSourceSchemas {
			schemaVersions[providerAddr][dataSourceType] = dataSourceSchema.Version
		}
	}

	return schemaVersions, nil
}
