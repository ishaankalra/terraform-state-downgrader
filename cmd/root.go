// Copyright (c) 2024 Ishaan Kalra
// SPDX-License-Identifier: MIT

package cmd

import (
	"github.com/spf13/cobra"
)

var (
	configDir string
	stateFile string
	backupFile string
)

var rootCmd = &cobra.Command{
	Use:   "terraform-state-downgrade",
	Short: "Downgrade Terraform state schema versions",
	Long: `terraform-state-downgrade is a tool that helps you downgrade Terraform state
when you've switched to an older provider version.

It works by:
1. Analyzing your configuration to detect schema version mismatches
2. Re-importing resources from the cloud provider
3. Updating the state file with the correct schema versions

Example:
  terraform-state-downgrade plan
  terraform-state-downgrade apply`,
}

func init() {
	rootCmd.PersistentFlags().StringVar(&configDir, "config-dir", ".", "Terraform configuration directory")
	rootCmd.PersistentFlags().StringVar(&stateFile, "state-file", "terraform.tfstate", "Path to state file")
	rootCmd.PersistentFlags().StringVar(&backupFile, "backup", "", "Backup file path (default: auto-generated)")
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}