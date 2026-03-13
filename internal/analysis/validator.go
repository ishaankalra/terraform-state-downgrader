// Copyright (c) 2024 Ishaan Kalra
// SPDX-License-Identifier: MIT

package analysis

import (
	"fmt"

	"github.com/ishaankalra/terraform-state-downgrader/internal/config"
)

// ValidateResourceSchema compares resource state attributes against provider schema
// Returns a list of schema validation issues found
func ValidateResourceSchema(
	attributes map[string]interface{},
	schema config.SchemaDefinition,
	resourceType string,
) []string {
	var issues []string

	// Get the block structure from schema
	block := schema.Block
	if block == nil {
		// If Block is nil, we can't validate
		return issues
	}

	// Get attributes definition from block
	schemaAttributes, ok := block["attributes"].(map[string]interface{})
	if !ok {
		// No attributes defined in schema
		return issues
	}

	// Check each attribute in the state against schema
	for attrName, attrValue := range attributes {
		// Skip special Terraform attributes
		if isSpecialAttribute(attrName) {
			continue
		}

		// Check if attribute exists in schema
		schemaAttr, exists := schemaAttributes[attrName]
		if !exists {
			issues = append(issues, fmt.Sprintf("attribute '%s' not found in provider schema", attrName))
			continue
		}

		// Validate attribute type if schema provides type information
		if err := validateAttributeType(attrName, attrValue, schemaAttr); err != nil {
			issues = append(issues, err.Error())
		}
	}

	// Check for missing required attributes
	for attrName, schemaAttr := range schemaAttributes {
		attrDef, ok := schemaAttr.(map[string]interface{})
		if !ok {
			continue
		}

		// Check if attribute is required
		required, _ := attrDef["required"].(bool)
		if required {
			if _, exists := attributes[attrName]; !exists {
				issues = append(issues, fmt.Sprintf("required attribute '%s' missing from state", attrName))
			}
		}
	}

	return issues
}

// isSpecialAttribute returns true for Terraform-managed attributes that should not be validated
func isSpecialAttribute(name string) bool {
	specialAttrs := map[string]bool{
		"id":         true, // Resource ID
		"timeouts":   true, // Timeout configuration
		"depends_on": true, // Dependencies
	}
	return specialAttrs[name]
}

// validateAttributeType checks if the attribute value type matches schema expectations
func validateAttributeType(attrName string, attrValue interface{}, schemaAttr interface{}) error {
	attrDef, ok := schemaAttr.(map[string]interface{})
	if !ok {
		return nil // Can't validate if schema attribute is not a map
	}

	// Get the type from schema
	attrType, ok := attrDef["type"]
	if !ok {
		return nil // No type information in schema
	}

	// Handle different schema type formats
	switch schemaType := attrType.(type) {
	case string:
		// Simple type like "string", "number", "bool"
		return validateSimpleType(attrName, attrValue, schemaType)
	case []interface{}:
		// Complex type like ["list", "string"] or ["set", "number"]
		if len(schemaType) > 0 {
			if containerType, ok := schemaType[0].(string); ok {
				return validateContainerType(attrName, attrValue, containerType)
			}
		}
	case map[string]interface{}:
		// Complex nested type definition
		// Could be ["map", "string"] or ["object", {...}]
		// For now, skip complex validation
		return nil
	}

	return nil
}

// validateSimpleType validates simple types like string, number, bool
func validateSimpleType(attrName string, value interface{}, expectedType string) error {
	switch expectedType {
	case "string":
		if _, ok := value.(string); !ok {
			return fmt.Errorf("attribute '%s' should be string, got %T", attrName, value)
		}
	case "number":
		switch value.(type) {
		case float64, int, int64:
			return nil
		default:
			return fmt.Errorf("attribute '%s' should be number, got %T", attrName, value)
		}
	case "bool":
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("attribute '%s' should be bool, got %T", attrName, value)
		}
	}
	return nil
}

// validateContainerType validates container types like list, set, map
func validateContainerType(attrName string, value interface{}, containerType string) error {
	switch containerType {
	case "list", "set":
		if _, ok := value.([]interface{}); !ok {
			return fmt.Errorf("attribute '%s' should be %s, got %T", attrName, containerType, value)
		}
	case "map":
		if _, ok := value.(map[string]interface{}); !ok {
			return fmt.Errorf("attribute '%s' should be map, got %T", attrName, value)
		}
	}
	return nil
}