// Copyright (c) 2024 Ishaan Kalra
// SPDX-License-Identifier: MIT

package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// ReadState reads and parses a Terraform state file
func ReadState(path string) (*State, error) {
	// Make path absolute
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Read file
	data, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read state file: %w", err)
	}

	// Parse JSON
	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to parse state JSON: %w", err)
	}

	return &state, nil
}