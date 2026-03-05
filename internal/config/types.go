// Copyright (c) 2024 Ishaan Kalra
// SPDX-License-Identifier: MIT

package config

// LockFile represents parsed .terraform.lock.hcl
type LockFile struct {
	Providers map[string]ProviderLock // provider address → version info
}

// ProviderLock represents a provider entry in the lock file
type ProviderLock struct {
	Version     string
	Constraints string
	Hashes      []string
}

// ProvidersSchemaOutput represents terraform providers schema -json output
type ProvidersSchemaOutput struct {
	FormatVersion   string                     `json:"format_version"`
	ProviderSchemas map[string]ProviderSchema  `json:"provider_schemas"`
}

// ProviderSchema represents a provider's schema
type ProviderSchema struct {
	Provider          SchemaDefinition            `json:"provider"`
	ResourceSchemas   map[string]SchemaDefinition `json:"resource_schemas"`
	DataSourceSchemas map[string]SchemaDefinition `json:"data_source_schemas"`
}

// SchemaDefinition represents a resource or data source schema
type SchemaDefinition struct {
	Version int64                  `json:"version"`
	Block   map[string]interface{} `json:"block"`
}