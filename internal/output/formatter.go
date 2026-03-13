// Copyright (c) 2024 Ishaan Kalra
// SPDX-License-Identifier: MIT

package output

import (
	"fmt"
	"sort"

	"github.com/ishaankalra/terraform-state-downgrader/internal/analysis"
	"github.com/ishaankalra/terraform-state-downgrader/internal/config"
	"github.com/ishaankalra/terraform-state-downgrader/internal/state"
)

// DisplayPlan displays the plan output in a human-readable format
func DisplayPlan(lockFile *config.LockFile, stateData *state.State, mismatches []analysis.Mismatch) {
	// Display providers from lock file
	fmt.Println("Providers from lock file:")
	for providerAddr, providerLock := range lockFile.Providers {
		fmt.Printf("  • %s v%s\n", providerAddr, providerLock.Version)
	}
	fmt.Println()

	// Display resource count
	managedCount := 0
	for _, resource := range stateData.Resources {
		if resource.Mode == "managed" {
			managedCount += len(resource.Instances)
		}
	}
	fmt.Printf("Resources analyzed: %d\n\n", managedCount)

	// Display mismatches
	if len(mismatches) == 0 {
		fmt.Println("✓ No schema issues found!")
		fmt.Println("  All resources are in sync with provider schemas.")
		return
	}

	// Count different types of issues
	versionMismatchCount := 0
	schemaIssueCount := 0
	for _, m := range mismatches {
		if m.HasVersionMismatch {
			versionMismatchCount++
		}
		if m.HasSchemaIssues {
			schemaIssueCount++
		}
	}

	fmt.Printf("Resources with issues: %d total", len(mismatches))
	if versionMismatchCount > 0 && schemaIssueCount > 0 {
		fmt.Printf(" (%d version mismatches, %d schema issues)\n\n", versionMismatchCount, schemaIssueCount)
	} else if versionMismatchCount > 0 {
		fmt.Printf(" (%d version mismatches)\n\n", versionMismatchCount)
	} else if schemaIssueCount > 0 {
		fmt.Printf(" (%d schema issues)\n\n", schemaIssueCount)
	} else {
		fmt.Println()
	}

	// Group mismatches by provider
	byProvider := make(map[string][]analysis.Mismatch)
	for _, mismatch := range mismatches {
		byProvider[mismatch.ProviderAddress] = append(byProvider[mismatch.ProviderAddress], mismatch)
	}

	// Sort provider addresses for consistent output
	var providerAddrs []string
	for addr := range byProvider {
		providerAddrs = append(providerAddrs, addr)
	}
	sort.Strings(providerAddrs)

	// Display mismatches grouped by provider
	for _, providerAddr := range providerAddrs {
		providerMismatches := byProvider[providerAddr]

		// Get provider version from lock file
		providerVersion := "unknown"
		if lock, ok := lockFile.Providers[providerAddr]; ok {
			providerVersion = lock.Version
		}

		fmt.Printf("%s:\n", providerAddr)
		if providerVersion != "unknown" {
			fmt.Printf("  Version: %s\n\n", providerVersion)
		}

		for _, mismatch := range providerMismatches {
			fmt.Printf("  • %s\n", mismatch.ResourceAddress)

			// Display version info if there's a version mismatch
			if mismatch.HasVersionMismatch {
				fmt.Printf("    State schema: v%d → Target schema: v%d\n",
					mismatch.StateVersion, mismatch.TargetVersion)
			} else {
				fmt.Printf("    State schema: v%d (matches provider)\n", mismatch.StateVersion)
			}

			if mismatch.ResourceID != "" {
				fmt.Printf("    Resource ID: %s\n", mismatch.ResourceID)
			}

			// Display action based on issue type
			if mismatch.HasVersionMismatch && mismatch.HasSchemaIssues {
				if mismatch.StateVersion > mismatch.TargetVersion {
					fmt.Printf("    ⚠️  DOWNGRADE REQUIRED (v%d → v%d) + SCHEMA VALIDATION ISSUES\n",
						mismatch.StateVersion, mismatch.TargetVersion)
				} else {
					fmt.Printf("    ⚠️  UPGRADE AVAILABLE (v%d → v%d) + SCHEMA VALIDATION ISSUES\n",
						mismatch.StateVersion, mismatch.TargetVersion)
				}
			} else if mismatch.HasVersionMismatch {
				if mismatch.StateVersion > mismatch.TargetVersion {
					fmt.Printf("    ⚠️  DOWNGRADE REQUIRED (v%d → v%d)\n",
						mismatch.StateVersion, mismatch.TargetVersion)
				} else {
					fmt.Printf("    ℹ️  UPGRADE AVAILABLE (v%d → v%d)\n",
						mismatch.StateVersion, mismatch.TargetVersion)
				}
			} else if mismatch.HasSchemaIssues {
				fmt.Printf("    ⚠️  SCHEMA VALIDATION ISSUES DETECTED\n")
			}

			// Display schema issues if present
			if mismatch.HasSchemaIssues {
				fmt.Printf("    Schema validation issues:\n")
				for _, issue := range mismatch.SchemaIssues {
					fmt.Printf("      - %s\n", issue)
				}
			}

			// Display action based on issue type
			if mismatch.HasVersionMismatch {
				fmt.Println("    Action: Remove from state and re-import from cloud provider")
			} else {
				fmt.Println("    Action: Refresh from cloud provider (in-place)")
			}

			// Display timeouts if present
			if len(mismatch.Timeouts) > 0 {
				fmt.Print("    Timeouts: ")
				first := true
				for key, value := range mismatch.Timeouts {
					if !first {
						fmt.Print(", ")
					}
					fmt.Printf("%s=%v", key, value)
					first = false
				}
				fmt.Println(" (preserved)")
			}

			fmt.Println()
		}
	}

	// Summary
	unchangedCount := managedCount - len(mismatches)
	fmt.Printf("Summary: %d to downgrade, %d unchanged\n", len(mismatches), unchangedCount)

	if len(mismatches) > 0 {
		fmt.Println("\nTo apply these changes:")
		fmt.Println("  terraform-state-downgrader apply")
	}
}
