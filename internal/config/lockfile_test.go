// Copyright (c) 2024 Ishaan Kalra
// SPDX-License-Identifier: MIT

package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseLockFile_SingleProvider(t *testing.T) {
	tmpDir := t.TempDir()
	lockPath := filepath.Join(tmpDir, ".terraform.lock.hcl")

	mockLockFile := `# This file is maintained automatically by "terraform init".
# Manual edits may be lost in future updates.

provider "registry.terraform.io/hashicorp/aws" {
  version     = "3.74.0"
  constraints = "~> 3.74"
  hashes = [
    "h1:xXYZ123",
    "h1:abc456"
  ]
}
`

	err := os.WriteFile(lockPath, []byte(mockLockFile), 0644)
	if err != nil {
		t.Fatalf("Failed to create lock file: %v", err)
	}

	// Parse the lock file
	lockFile, err := ParseLockFile(tmpDir)
	if err != nil {
		t.Fatalf("ParseLockFile failed: %v", err)
	}

	// Assertions
	if len(lockFile.Providers) != 1 {
		t.Fatalf("Expected 1 provider, got %d", len(lockFile.Providers))
	}

	providerAddr := "registry.terraform.io/hashicorp/aws"
	provider, ok := lockFile.Providers[providerAddr]
	if !ok {
		t.Fatalf("Provider %s not found", providerAddr)
	}

	if provider.Version != "3.74.0" {
		t.Errorf("Expected version 3.74.0, got %s", provider.Version)
	}

	if provider.Constraints != "~> 3.74" {
		t.Errorf("Expected constraints ~> 3.74, got %s", provider.Constraints)
	}

	if len(provider.Hashes) != 2 {
		t.Fatalf("Expected 2 hashes, got %d", len(provider.Hashes))
	}

	if provider.Hashes[0] != "h1:xXYZ123" {
		t.Errorf("Expected hash h1:xXYZ123, got %s", provider.Hashes[0])
	}
}

func TestParseLockFile_MultipleProviders(t *testing.T) {
	tmpDir := t.TempDir()
	lockPath := filepath.Join(tmpDir, ".terraform.lock.hcl")

	mockLockFile := `provider "registry.terraform.io/hashicorp/aws" {
  version     = "3.74.0"
  constraints = "~> 3.74"
}

provider "registry.terraform.io/hashicorp/azurerm" {
  version     = "2.50.0"
  constraints = "~> 2.50"
}

provider "registry.terraform.io/hashicorp/random" {
  version = "3.5.1"
}
`

	os.WriteFile(lockPath, []byte(mockLockFile), 0644)

	lockFile, err := ParseLockFile(tmpDir)
	if err != nil {
		t.Fatalf("ParseLockFile failed: %v", err)
	}

	if len(lockFile.Providers) != 3 {
		t.Fatalf("Expected 3 providers, got %d", len(lockFile.Providers))
	}

	// Check AWS provider
	aws := lockFile.Providers["registry.terraform.io/hashicorp/aws"]
	if aws.Version != "3.74.0" {
		t.Errorf("AWS version: expected 3.74.0, got %s", aws.Version)
	}

	// Check Azure provider
	azure := lockFile.Providers["registry.terraform.io/hashicorp/azurerm"]
	if azure.Version != "2.50.0" {
		t.Errorf("Azure version: expected 2.50.0, got %s", azure.Version)
	}

	// Check Random provider (no constraints)
	random := lockFile.Providers["registry.terraform.io/hashicorp/random"]
	if random.Version != "3.5.1" {
		t.Errorf("Random version: expected 3.5.1, got %s", random.Version)
	}
	if random.Constraints != "" {
		t.Errorf("Random constraints: expected empty, got %s", random.Constraints)
	}
}

func TestParseLockFile_FileNotFound(t *testing.T) {
	tmpDir := t.TempDir()

	_, err := ParseLockFile(tmpDir)
	if err == nil {
		t.Fatal("Expected error for missing lock file, got nil")
	}

	expectedMsg := "lock file not found"
	if err.Error()[:len(expectedMsg)] != expectedMsg {
		t.Errorf("Expected error message to start with '%s', got: %s", expectedMsg, err.Error())
	}
}

func TestParseLockFile_InvalidHCL(t *testing.T) {
	tmpDir := t.TempDir()
	lockPath := filepath.Join(tmpDir, ".terraform.lock.hcl")

	invalidHCL := `this is not valid HCL {{{`
	os.WriteFile(lockPath, []byte(invalidHCL), 0644)

	_, err := ParseLockFile(tmpDir)
	if err == nil {
		t.Fatal("Expected error for invalid HCL, got nil")
	}
}

func TestParseLockFile_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	lockPath := filepath.Join(tmpDir, ".terraform.lock.hcl")

	emptyContent := `# Empty lock file`
	os.WriteFile(lockPath, []byte(emptyContent), 0644)

	_, err := ParseLockFile(tmpDir)
	if err == nil {
		t.Fatal("Expected error for empty lock file (no providers), got nil")
	}

	expectedMsg := "no providers found in lock file"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error '%s', got: %s", expectedMsg, err.Error())
	}
}

func TestParseLockFile_NoVersion(t *testing.T) {
	tmpDir := t.TempDir()
	lockPath := filepath.Join(tmpDir, ".terraform.lock.hcl")

	// Provider block without version attribute
	noVersionHCL := `provider "registry.terraform.io/hashicorp/aws" {
  constraints = "~> 3.74"
}
`
	os.WriteFile(lockPath, []byte(noVersionHCL), 0644)

	lockFile, err := ParseLockFile(tmpDir)
	if err != nil {
		t.Fatalf("ParseLockFile failed: %v", err)
	}

	provider := lockFile.Providers["registry.terraform.io/hashicorp/aws"]
	if provider.Version != "" {
		t.Errorf("Expected empty version, got %s", provider.Version)
	}

	// Should still have constraints
	if provider.Constraints != "~> 3.74" {
		t.Errorf("Expected constraints ~> 3.74, got %s", provider.Constraints)
	}
}

func TestParseLockFile_WithComments(t *testing.T) {
	tmpDir := t.TempDir()
	lockPath := filepath.Join(tmpDir, ".terraform.lock.hcl")

	hclWithComments := `# This file is maintained automatically
# Do not edit manually

provider "registry.terraform.io/hashicorp/random" {
  # Version pinned for stability
  version = "3.5.1"

  # Constraint from terraform block
  constraints = "~> 3.5"

  # Platform-specific hashes
  hashes = [
    "h1:abc123", # darwin_amd64
    "h1:def456", # linux_amd64
  ]
}
`
	os.WriteFile(lockPath, []byte(hclWithComments), 0644)

	lockFile, err := ParseLockFile(tmpDir)
	if err != nil {
		t.Fatalf("ParseLockFile failed: %v", err)
	}

	if len(lockFile.Providers) != 1 {
		t.Fatalf("Expected 1 provider, got %d", len(lockFile.Providers))
	}

	provider := lockFile.Providers["registry.terraform.io/hashicorp/random"]
	if provider.Version != "3.5.1" {
		t.Errorf("Expected version 3.5.1, got %s", provider.Version)
	}

	if len(provider.Hashes) != 2 {
		t.Errorf("Expected 2 hashes, got %d", len(provider.Hashes))
	}
}
