// Copyright (c) 2024 Ishaan Kalra
// SPDX-License-Identifier: MIT

package provider

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/ishaankalra/terraform-state-downgrader/internal/analysis"
	"github.com/ishaankalra/terraform-state-downgrader/internal/state"
)

// ReimportResources re-imports resources from cloud provider to update schema versions
func ReimportResources(
	configDir string,
	stateData *state.State,
	mismatches []analysis.Mismatch,
	schemaVersions map[string]map[string]int64,
) error {
	// Group mismatches by provider for loading
	providersByAddr := make(map[string]bool)
	for _, mismatch := range mismatches {
		providersByAddr[mismatch.ProviderAddress] = true
	}

	fmt.Printf("  Providers needed: %d\n", len(providersByAddr))
	for providerAddr := range providersByAddr {
		fmt.Printf("  • %s\n", providerAddr)
	}
	fmt.Println()

	// Phase 1: Remove all mismatched resources from state
	fmt.Printf("Removing resources from state (%d total):\n", len(mismatches))
	for idx, mismatch := range mismatches {
		if err := removeFromState(configDir, mismatch.ResourceAddress); err != nil {
			fmt.Printf("  [%d/%d] ✗ %s - %v\n", idx+1, len(mismatches), mismatch.ResourceAddress, err)
			return fmt.Errorf("failed to remove %s from state: %w", mismatch.ResourceAddress, err)
		}
		fmt.Printf("  [%d/%d] ✓ %s removed\n", idx+1, len(mismatches), mismatch.ResourceAddress)
	}
	fmt.Println()

	// Phase 2: Create import.tf.json with import blocks
	importFile := filepath.Join(configDir, "import.tf.json")
	if err := createImportFile(importFile, mismatches); err != nil {
		return fmt.Errorf("failed to create import.tf.json: %w", err)
	}
	defer os.Remove(importFile) // Clean up import file after we're done

	fmt.Printf("Created %s with %d import blocks\n", importFile, len(mismatches))
	fmt.Println()

	// Phase 3: Run terraform refresh to import all resources at once
	fmt.Printf("Running terraform refresh to import resources...\n")
	refreshCmd := exec.Command("terraform", "refresh")
	refreshCmd.Dir = configDir
	output, err := refreshCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("terraform refresh failed: %w\nOutput: %s", err, string(output))
	}

	fmt.Printf("✓ Successfully imported %d resources\n", len(mismatches))
	return nil
}

// removeFromState removes a resource from the Terraform state
func removeFromState(configDir, resourceAddress string) error {
	rmCmd := exec.Command("terraform", "state", "rm", resourceAddress)
	rmCmd.Dir = configDir
	output, err := rmCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("terraform state rm failed: %w\nOutput: %s", err, string(output))
	}
	return nil
}

// createImportFile creates an import.tf.json file with import blocks for all mismatches
func createImportFile(filePath string, mismatches []analysis.Mismatch) error {
	// Build import blocks structure for Terraform JSON
	importBlocks := make([]map[string]interface{}, 0, len(mismatches))

	for _, mismatch := range mismatches {
		// Verify resource ID exists
		if mismatch.ResourceID == "" {
			return fmt.Errorf("resource %s has empty ID", mismatch.ResourceAddress)
		}

		importBlock := map[string]interface{}{
			"to": mismatch.ResourceAddress,
			"id": mismatch.ResourceID,
		}
		importBlocks = append(importBlocks, importBlock)
	}

	// Wrap in the terraform import structure
	importConfig := map[string]interface{}{
		"import": importBlocks,
	}

	// Write to import.tf.json
	data, err := json.MarshalIndent(importConfig, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal import config: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write import file: %w", err)
	}

	return nil
}
