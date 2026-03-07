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
	fmt.Printf("Removing %d resources from state...\n", len(mismatches))

	// Collect all resource addresses
	resourceAddresses := make([]string, len(mismatches))
	for idx, mismatch := range mismatches {
		resourceAddresses[idx] = mismatch.ResourceAddress
	}

	// Remove all resources in a single command
	if err := removeFromState(configDir, resourceAddresses...); err != nil {
		return fmt.Errorf("failed to remove resources from state: %w", err)
	}
	fmt.Printf("✓ Removed %d resources from state\n", len(mismatches))
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
	// Use -target flags to refresh only the specific resources that need to be imported
	fmt.Printf("Running terraform refresh to import resources...\n")

	// Build refresh command with target flags for each resource
	args := []string{"refresh"}
	for _, mismatch := range mismatches {
		args = append(args, "-target="+mismatch.ResourceAddress)
	}

	refreshCmd := exec.Command("terraform", args...)
	refreshCmd.Dir = configDir
	// Stream output in real-time
	refreshCmd.Stdout = os.Stdout
	refreshCmd.Stderr = os.Stderr

	if err := refreshCmd.Run(); err != nil {
		return fmt.Errorf("terraform refresh failed: %w", err)
	}

	fmt.Printf("\n✓ Successfully imported %d resources\n", len(mismatches))
	return nil
}

// removeFromState removes multiple resources from the Terraform state in a single command
func removeFromState(configDir string, resourceAddresses ...string) error {
	if len(resourceAddresses) == 0 {
		return nil
	}

	// Build command: terraform state rm <addr1> <addr2> <addr3> ...
	args := append([]string{"state", "rm"}, resourceAddresses...)
	rmCmd := exec.Command("terraform", args...)
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
