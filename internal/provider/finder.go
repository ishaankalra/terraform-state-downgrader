// Copyright (c) 2024 Ishaan Kalra
// SPDX-License-Identifier: MIT

package provider

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// FindProviderBinary finds the provider binary in .terraform/providers/
// Input: "registry.terraform.io/hashicorp/aws", "3.74.0"
// Returns: path to provider binary
func FindProviderBinary(configDir, providerAddr, version string) (string, error) {
	// Provider binaries are in: .terraform/providers/registry.terraform.io/namespace/name/version/os_arch/
	// Example: .terraform/providers/registry.terraform.io/hashicorp/aws/3.74.0/darwin_arm64/terraform-provider-aws_v3.74.0

	providersDir := filepath.Join(configDir, ".terraform", "providers")

	// Parse provider address
	parts := strings.Split(providerAddr, "/")
	if len(parts) < 3 {
		return "", fmt.Errorf("invalid provider address: %s", providerAddr)
	}

	hostname := parts[0]
	namespace := parts[1]
	name := parts[2]

	// Determine OS and architecture
	osArch := fmt.Sprintf("%s_%s", runtime.GOOS, runtime.GOARCH)

	// Build path to provider binary
	providerDir := filepath.Join(providersDir, hostname, namespace, name, version, osArch)

	// Check if directory exists
	if _, err := os.Stat(providerDir); os.IsNotExist(err) {
		return "", fmt.Errorf("provider binary not found at %s. Run 'terraform init' first", providerDir)
	}

	// Find the binary file
	entries, err := os.ReadDir(providerDir)
	if err != nil {
		return "", fmt.Errorf("failed to read provider directory: %w", err)
	}

	// Look for terraform-provider-* executable
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		filename := entry.Name()
		if strings.HasPrefix(filename, "terraform-provider-") {
			binaryPath := filepath.Join(providerDir, filename)

			// Verify it's executable
			info, err := os.Stat(binaryPath)
			if err != nil {
				continue
			}

			// Check if executable (on Unix systems)
			if runtime.GOOS != "windows" && info.Mode()&0111 == 0 {
				continue
			}

			return binaryPath, nil
		}
	}

	return "", fmt.Errorf("provider binary not found in %s", providerDir)
}