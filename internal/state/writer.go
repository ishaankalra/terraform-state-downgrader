// Copyright (c) 2024 Ishaan Kalra
// SPDX-License-Identifier: MIT

package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// WriteState writes a Terraform state file atomically
func WriteState(path string, state *State) error {
	// Make path absolute
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Marshal to JSON with indentation (Terraform uses 2-space indent)
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	// Add trailing newline (Terraform does this)
	data = append(data, '\n')

	// Write to temporary file first (atomic write)
	tmpPath := absPath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	// Rename to final path (atomic on POSIX systems)
	if err := os.Rename(tmpPath, absPath); err != nil {
		os.Remove(tmpPath) // Clean up temp file
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	return nil
}

// CreateBackup creates a backup copy of the state file
func CreateBackup(sourcePath, backupPath string) error {
	// Make paths absolute
	absSource, err := filepath.Abs(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to get absolute source path: %w", err)
	}

	absBackup, err := filepath.Abs(backupPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute backup path: %w", err)
	}

	// Read source
	data, err := os.ReadFile(absSource)
	if err != nil {
		return fmt.Errorf("failed to read source file: %w", err)
	}

	// Write backup
	if err := os.WriteFile(absBackup, data, 0644); err != nil {
		return fmt.Errorf("failed to write backup file: %w", err)
	}

	return nil
}