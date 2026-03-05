// Copyright (c) 2024 Ishaan Kalra
// SPDX-License-Identifier: MIT

package cmd

import (
	"github.com/spf13/cobra"
)

var (
	configDir  string
	backupFile string
)

var rootCmd = &cobra.Command{
	Use:   "terraform-state-downgrade",
	Short: "Downgrade Terraform state schema versions",
	Long: `terraform-state-downgrade is a tool that helps you downgrade Terraform state
when you've switched to an older provider version.

It works by:
1. Pulling state from your configured backend (supports all backends)
2. Analyzing your configuration to detect schema version mismatches
3. Re-importing resources from the cloud provider
4. State is automatically pushed back to the backend by Terraform

Example:
  terraform-state-downgrade plan
  terraform-state-downgrade apply`,
}

func init() {
	rootCmd.PersistentFlags().StringVar(&configDir, "config-dir", ".", "Terraform configuration directory")
	rootCmd.PersistentFlags().StringVar(&backupFile, "backup", "", "Backup file path (default: auto-generated)")
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}

// ExecuteWithArgs runs the root command with specified arguments
// This is useful for testing
func ExecuteWithArgs(args []string) error {
	rootCmd.SetArgs(args)
	return rootCmd.Execute()
}