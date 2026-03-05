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
		fmt.Println("✓ No schema version mismatches found!")
		fmt.Println("  All resources are in sync with provider schemas.")
		return
	}

	fmt.Printf("Schema version mismatches: %d resources\n\n", len(mismatches))

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
			fmt.Printf("    State schema: v%d → Target schema: v%d\n",
				mismatch.StateVersion, mismatch.TargetVersion)

			if mismatch.ResourceID != "" {
				fmt.Printf("    Resource ID: %s\n", mismatch.ResourceID)
			}

			// Display action
			if mismatch.StateVersion > mismatch.TargetVersion {
				fmt.Printf("    ⚠️  DOWNGRADE REQUIRED (v%d → v%d)\n",
					mismatch.StateVersion, mismatch.TargetVersion)
			} else {
				fmt.Printf("    ℹ️  UPGRADE AVAILABLE (v%d → v%d)\n",
					mismatch.StateVersion, mismatch.TargetVersion)
			}

			fmt.Println("    Action: Re-import from cloud provider")

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
