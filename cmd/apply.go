// Copyright (c) 2024 Ishaan Kalra
// SPDX-License-Identifier: MIT

package cmd

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/ishaankalra/terraform-state-downgrade/internal/analysis"
	"github.com/ishaankalra/terraform-state-downgrade/internal/config"
	"github.com/ishaankalra/terraform-state-downgrade/internal/provider"
	"github.com/ishaankalra/terraform-state-downgrade/internal/state"
	"github.com/spf13/cobra"
)

var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Apply the downgrade changes",
	Long: `Downgrades resources by re-importing them from the cloud provider.
Creates a backup before making any changes.`,
	RunE: runApply,
}

func init() {
	rootCmd.AddCommand(applyCmd)
}

func runApply(cmd *cobra.Command, args []string) error {
	startTime := time.Now()
	fmt.Println("terraform-state-downgrade apply")
	fmt.Println()

	// Step 1: Parse lock file
	fmt.Println("Analyzing configuration...")
	lockFile, err := config.ParseLockFile(configDir)
	if err != nil {
		return fmt.Errorf("failed to parse lock file: %w", err)
	}
	fmt.Printf("  ✓ Parsed .terraform.lock.hcl (%d providers)\n", len(lockFile.Providers))

	// Step 2: Pull state from backend
	fmt.Println("  ✓ Running: terraform state pull")
	stateData, stateBytes, err := state.PullState(configDir)
	if err != nil {
		return fmt.Errorf("failed to pull state: %w", err)
	}

	// Step 3: Get resource-to-provider mapping from terraform providers and state
	fmt.Println("  ✓ Running: terraform providers")
	resourceMapping, err := config.GetResourceProviderMappingFromState(configDir, stateData)
	if err != nil {
		return fmt.Errorf("failed to get resource-provider mapping: %w", err)
	}

	// Step 4: Get schema versions
	fmt.Println("  ✓ Running: terraform providers schema -json")
	schemaVersions, err := config.GetSchemaVersions(configDir)
	if err != nil {
		return fmt.Errorf("failed to get schema versions: %w", err)
	}

	// Step 5: Detect mismatches
	mismatches, err := analysis.DetectMismatches(stateData, resourceMapping, schemaVersions)
	if err != nil {
		return fmt.Errorf("failed to detect mismatches: %w", err)
	}

	if len(mismatches) == 0 {
		fmt.Println("\n✓ No resources need downgrade. State is already in sync!")
		return nil
	}

	// Step 6: Create backup from pulled state
	backupPath := backupFile
	if backupPath == "" {
		backupPath = filepath.Join(configDir, fmt.Sprintf("terraform.tfstate.backup-%d", time.Now().Unix()))
	}
	fmt.Printf("\nCreating backup: %s\n", backupPath)
	if err := state.CreateBackupFromBytes(stateBytes, backupPath); err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	// Step 7: Load providers and re-import resources
	fmt.Println("\nLoading providers...")
	if err := provider.ReimportResources(configDir, stateData, mismatches, schemaVersions); err != nil {
		return fmt.Errorf("failed to re-import resources: %w", err)
	}

	// Step 8: Summary
	elapsed := time.Since(startTime)
	fmt.Printf("\n✓ Success! %d resources downgraded\n", len(mismatches))
	fmt.Printf("  Backup: %s\n", backupPath)
	fmt.Printf("  Time: %.1fs\n", elapsed.Seconds())

	return nil
}