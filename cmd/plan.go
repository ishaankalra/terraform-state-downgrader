// Copyright (c) 2024 Ishaan Kalra
// SPDX-License-Identifier: MIT

package cmd

import (
	"fmt"

	"github.com/ishaankalra/terraform-state-downgrader/internal/analysis"
	"github.com/ishaankalra/terraform-state-downgrader/internal/config"
	"github.com/ishaankalra/terraform-state-downgrader/internal/output"
	"github.com/ishaankalra/terraform-state-downgrader/internal/state"
	"github.com/spf13/cobra"
)

var planCmd = &cobra.Command{
	Use:   "plan",
	Short: "Show what changes would be made",
	Long: `Analyzes your configuration and state to show which resources need
schema version downgrade. Does not modify any files.`,
	RunE: runPlan,
}

func init() {
	rootCmd.AddCommand(planCmd)
}

func runPlan(cmd *cobra.Command, args []string) error {
	fmt.Println("terraform-state-downgrader plan")
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
	stateData, _, err := state.PullState(configDir)
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
	fmt.Printf("  ✓ Running: terraform providers schema -json\n\n")
	schemaVersions, err := config.GetSchemaVersions(configDir)
	if err != nil {
		return fmt.Errorf("failed to get schema versions: %w", err)
	}

	// Step 5: Detect mismatches
	fmt.Println("Analyzing resources...")
	mismatches, err := analysis.DetectMismatches(stateData, resourceMapping, schemaVersions)
	if err != nil {
		return fmt.Errorf("failed to detect mismatches: %w", err)
	}

	// Step 6: Display plan
	output.DisplayPlan(lockFile, stateData, mismatches)

	return nil
}