// Copyright (c) 2024 Ishaan Kalra
// SPDX-License-Identifier: MIT

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	configDir  string
	backupFile string
	version    string
)

var rootCmd = &cobra.Command{
	Use:   "terraform-state-downgrader",
	Short: "Downgrade Terraform state schema versions",
	Long: `terraform-state-downgrader is a tool that helps you downgrade Terraform state
when you've switched to an older provider version.

It works by:
1. Pulling state from your configured backend (supports all backends)
2. Analyzing your configuration to detect schema version mismatches
3. Re-importing resources from the cloud provider
4. State is automatically pushed back to the backend by Terraform

Example:
  terraform-state-downgrader plan
  terraform-state-downgrader apply`,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("terraform-state-downgrader %s\n", version)
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&configDir, "config-dir", ".", "Terraform configuration directory")
	rootCmd.PersistentFlags().StringVar(&backupFile, "backup", "", "Backup file path (default: auto-generated)")
	rootCmd.AddCommand(versionCmd)
}

// SetVersion sets the version string
func SetVersion(v string) {
	version = v
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