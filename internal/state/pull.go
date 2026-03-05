// Copyright (c) 2024 Ishaan Kalra
// SPDX-License-Identifier: MIT

package state

import (
	"encoding/json"
	"fmt"
	"os/exec"
)

// PullState pulls Terraform state from the configured backend using terraform state pull
// Returns the parsed state and the raw JSON bytes for backup purposes
func PullState(configDir string) (*State, []byte, error) {
	cmd := exec.Command("terraform", "state", "pull")
	cmd.Dir = configDir

	output, err := cmd.Output()
	if err != nil {
		return nil, nil, fmt.Errorf("terraform state pull failed: %w", err)
	}

	// Parse JSON
	var state State
	if err := json.Unmarshal(output, &state); err != nil {
		return nil, nil, fmt.Errorf("failed to parse state JSON: %w", err)
	}

	return &state, output, nil
}