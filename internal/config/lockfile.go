// Copyright (c) 2024 Ishaan Kalra
// SPDX-License-Identifier: MIT

package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclparse"
)

// ParseLockFile parses .terraform.lock.hcl and extracts provider versions
func ParseLockFile(configDir string) (*LockFile, error) {
	lockPath := filepath.Join(configDir, ".terraform.lock.hcl")

	// Check if file exists
	if _, err := os.Stat(lockPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("lock file not found at %s. Run 'terraform init' first", lockPath)
	}

	// Read the file
	data, err := os.ReadFile(lockPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read lock file: %w", err)
	}

	// Parse HCL
	parser := hclparse.NewParser()
	file, diags := parser.ParseHCL(data, lockPath)
	if diags.HasErrors() {
		return nil, fmt.Errorf("failed to parse HCL: %s", diags.Error())
	}

	// Extract provider blocks
	lockFile := &LockFile{
		Providers: make(map[string]ProviderLock),
	}

	content, _, diags := file.Body.PartialContent(&hcl.BodySchema{
		Blocks: []hcl.BlockHeaderSchema{
			{
				Type:       "provider",
				LabelNames: []string{"name"},
			},
		},
	})
	if diags.HasErrors() {
		return nil, fmt.Errorf("failed to extract content: %s", diags.Error())
	}

	// Process each provider block
	for _, block := range content.Blocks {
		providerAddr := block.Labels[0]

		attrs, diags := block.Body.JustAttributes()
		if diags.HasErrors() {
			continue
		}

		var providerLock ProviderLock

		// Extract version
		if versionAttr, ok := attrs["version"]; ok {
			var version string
			diags := gohcl.DecodeExpression(versionAttr.Expr, nil, &version)
			if diags.HasErrors() {
				continue
			}
			providerLock.Version = version
		}

		// Extract constraints
		if constraintsAttr, ok := attrs["constraints"]; ok {
			var constraints string
			gohcl.DecodeExpression(constraintsAttr.Expr, nil, &constraints)
			providerLock.Constraints = constraints
		}

		// Extract hashes
		if hashesAttr, ok := attrs["hashes"]; ok {
			var hashes []string
			gohcl.DecodeExpression(hashesAttr.Expr, nil, &hashes)
			providerLock.Hashes = hashes
		}

		lockFile.Providers[providerAddr] = providerLock
	}

	if len(lockFile.Providers) == 0 {
		return nil, fmt.Errorf("no providers found in lock file")
	}

	return lockFile, nil
}