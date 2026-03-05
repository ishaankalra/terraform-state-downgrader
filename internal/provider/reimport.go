// Copyright (c) 2024 Ishaan Kalra
// SPDX-License-Identifier: MIT

package provider

import (
	"fmt"
	"os/exec"
	"time"

	"github.com/ishaankalra/terraform-state-downgrade/internal/analysis"
	"github.com/ishaankalra/terraform-state-downgrade/internal/state"
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

	// Process each mismatch
	fmt.Printf("Re-importing resources (%d total):\n", len(mismatches))

	for idx, mismatch := range mismatches {
		start := time.Now()

		// Re-import the resource using terraform import
		// This is simpler than loading providers directly via gRPC
		err := reimportResource(configDir, stateData, mismatch, schemaVersions)

		elapsed := time.Since(start).Seconds()

		if err != nil {
			fmt.Printf("  [%d/%d] ✗ %s (%.1fs) - %v\n",
				idx+1, len(mismatches), mismatch.ResourceAddress, elapsed, err)
			return fmt.Errorf("failed to re-import %s: %w", mismatch.ResourceAddress, err)
		}

		fmt.Printf("  [%d/%d] ✓ %s (%.1fs)\n",
			idx+1, len(mismatches), mismatch.ResourceAddress, elapsed)
	}

	return nil
}

// reimportResource re-imports a single resource
func reimportResource(
	configDir string,
	stateData *state.State,
	mismatch analysis.Mismatch,
	schemaVersions map[string]map[string]int64,
) error {
	// Verify resource ID exists
	if mismatch.ResourceID == "" {
		return fmt.Errorf("resource ID is empty")
	}

	// Build import address
	importAddr := mismatch.ResourceAddress

	// Remove from state first
	rmCmd := exec.Command("terraform", "state", "rm", importAddr)
	rmCmd.Dir = configDir
	if err := rmCmd.Run(); err != nil {
		return fmt.Errorf("failed to remove from state: %w", err)
	}

	// Import with new schema version
	importCmd := exec.Command("terraform", "import", importAddr, mismatch.ResourceID)
	importCmd.Dir = configDir

	// Capture output for debugging
	output, err := importCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("terraform import failed: %w\nOutput: %s", err, string(output))
	}

	// Update the state in memory with new schema version and timeouts
	updateStateWithMismatch(stateData, mismatch, schemaVersions)

	return nil
}

// updateStateWithMismatch updates state after re-import to preserve timeouts
func updateStateWithMismatch(
	stateData *state.State,
	mismatch analysis.Mismatch,
	schemaVersions map[string]map[string]int64,
) {
	// Find the resource in state
	for i := range stateData.Resources {
		resource := &stateData.Resources[i]

		if resource.Type != mismatch.ResourceType || resource.Name != mismatch.ResourceName {
			continue
		}

		// Update the specific instance
		if mismatch.InstanceIndex < len(resource.Instances) {
			instance := &resource.Instances[mismatch.InstanceIndex]

			// Update schema version
			targetVersion := schemaVersions[mismatch.ProviderAddress][mismatch.ResourceType]
			instance.SchemaVersion = int(targetVersion)

			// Merge timeouts back if they existed
			if len(mismatch.Timeouts) > 0 {
				if instance.Attributes == nil {
					instance.Attributes = make(map[string]interface{})
				}
				instance.Attributes["timeouts"] = mismatch.Timeouts
			}
		}

		break
	}
}
