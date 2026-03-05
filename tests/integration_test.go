// Copyright (c) 2024 Ishaan Kalra
// SPDX-License-Identifier: MIT

package tests

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ishaankalra/terraform-state-downgrade/cmd"
	"github.com/ishaankalra/terraform-state-downgrade/internal/state"
)

// verifySchemaVersions strictly checks that all resources have the expected schema version
func verifySchemaVersions(t *testing.T, stateFile string, expectedVersion int) {
	data, err := os.ReadFile(stateFile)
	if err != nil {
		t.Fatalf("Failed to read state file: %v", err)
	}

	var stateData state.State
	if err := json.Unmarshal(data, &stateData); err != nil {
		t.Fatalf("Failed to parse state JSON: %v", err)
	}

	if len(stateData.Resources) == 0 {
		t.Fatalf("No resources found in state")
	}

	for _, resource := range stateData.Resources {
		// Skip data sources
		if resource.Mode != "managed" {
			continue
		}

		for i, instance := range resource.Instances {
			if instance.SchemaVersion != expectedVersion {
				t.Errorf("Resource %s.%s[%d] has schema_version %d, expected %d",
					resource.Type, resource.Name, i, instance.SchemaVersion, expectedVersion)
			}
		}
	}
}

// TestIntegration_Plan tests the plan command detects schema mismatches
func TestIntegration_Plan(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	projectRoot, err := filepath.Abs("..")
	if err != nil {
		t.Fatalf("Failed to get project root: %v", err)
	}

	testDir := filepath.Join(projectRoot, "tests", "random_provider")
	versionsFile := filepath.Join(testDir, "versions.tf")

	// Cleanup function
	cleanup := func() {
		os.Remove(versionsFile)
		os.RemoveAll(filepath.Join(testDir, ".terraform"))
		os.Remove(filepath.Join(testDir, ".terraform.lock.hcl"))
		os.Remove(filepath.Join(testDir, "terraform.tfstate"))
		os.Remove(filepath.Join(testDir, "terraform.tfstate.backup"))
	}
	defer cleanup()

	// Step 1: Create versions.tf with v3.5.1
	t.Run("setup_v3.5.1", func(t *testing.T) {
		versionsContent := `terraform {
  required_providers {
    random = {
      source  = "hashicorp/random"
      version = "~> 3.5.1"
    }
  }
}
`
		err := os.WriteFile(versionsFile, []byte(versionsContent), 0644)
		if err != nil {
			t.Fatalf("Failed to create versions.tf: %v", err)
		}

		// Run terraform init
		cmd := exec.Command("terraform", "init")
		cmd.Dir = testDir
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("terraform init failed: %v\nOutput: %s", err, output)
		}

		// Run terraform apply
		cmd = exec.Command("terraform", "apply", "-auto-approve")
		cmd.Dir = testDir
		output, err = cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("terraform apply failed: %v\nOutput: %s", err, output)
		}

		t.Logf("Successfully created state with v3.5.1")
	})

	// Step 2: Verify state has resources with schema_version > 0
	t.Run("verify_v3.5.1_state", func(t *testing.T) {
		stateFile := filepath.Join(testDir, "terraform.tfstate")
		data, err := os.ReadFile(stateFile)
		if err != nil {
			t.Fatalf("Failed to read state: %v", err)
		}

		stateContent := string(data)
		if !strings.Contains(stateContent, `"schema_version": 2`) {
			t.Errorf("Expected random_string to have schema_version 2")
		}
		if !strings.Contains(stateContent, `"schema_version": 3`) {
			t.Errorf("Expected random_password to have schema_version 3")
		}

		t.Logf("State contains resources with schema_version 2 and 3")
	})

	// Step 3: Downgrade to v3.1.0 (older version with lower schema versions)
	t.Run("downgrade_to_v3.1.0", func(t *testing.T) {
		versionsContent := `terraform {
  required_providers {
    random = {
      source  = "hashicorp/random"
      version = "~> 3.1.0"
    }
  }
}
`
		err := os.WriteFile(versionsFile, []byte(versionsContent), 0644)
		if err != nil {
			t.Fatalf("Failed to update versions.tf: %v", err)
		}

		// Run terraform init -upgrade
		cmd := exec.Command("terraform", "init", "-upgrade")
		cmd.Dir = testDir
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("terraform init -upgrade failed: %v\nOutput: %s", err, output)
		}

		t.Logf("Successfully downgraded to v3.1.0")
	})

	// Step 4: Verify terraform plan fails with schema mismatch
	t.Run("terraform_plan_fails_with_mismatch", func(t *testing.T) {
		cmd := exec.Command("terraform", "plan")
		cmd.Dir = testDir
		output, err := cmd.CombinedOutput()

		if err == nil {
			t.Errorf("Expected terraform plan to fail with schema mismatch")
		}

		outputStr := string(output)
		if !strings.Contains(outputStr, "newer provider version") {
			t.Errorf("Expected 'newer provider version' error, got:\n%s", outputStr)
		}

		t.Logf("Confirmed: terraform detects schema version mismatch")
	})

	// Step 5: Test the downgrade tool - should now work even with schema mismatch!
	t.Run("downgrade_tool_detects_mismatches", func(t *testing.T) {
		stateFilePath := filepath.Join(testDir, "terraform.tfstate")

		// Capture stdout
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		// Run plan command
		err := cmd.ExecuteWithArgs([]string{
			"plan",
			"--config-dir", testDir,
			"--state-file", stateFilePath,
		})

		// Restore stdout
		w.Close()
		os.Stdout = oldStdout

		var buf bytes.Buffer
		io.Copy(&buf, r)
		outputStr := buf.String()

		t.Logf("Tool output:\n%s", outputStr)

		// The tool should now succeed (no longer depends on terraform plan/show)
		if err != nil {
			t.Errorf("Tool should succeed with new approach: %v\nOutput: %s", err, outputStr)
			return
		}

		// Verify it detected the schema mismatches
		if !strings.Contains(outputStr, "Schema version mismatches") {
			t.Errorf("Expected tool to detect schema version mismatches")
		}

		// Should detect both random_string and random_password
		if !strings.Contains(outputStr, "random_string") || !strings.Contains(outputStr, "random_password") {
			t.Errorf("Expected tool to report random_string and random_password mismatches")
		}

		t.Logf("✓ Tool successfully detected mismatches without running terraform plan!")
	})
}

// TestIntegration_Apply tests the apply command fixes schema mismatches
func TestIntegration_Apply(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	projectRoot, err := filepath.Abs("..")
	if err != nil {
		t.Fatalf("Failed to get project root: %v", err)
	}

	testDir := filepath.Join(projectRoot, "tests", "random_provider")
	versionsFile := filepath.Join(testDir, "versions.tf")

	// Cleanup function
	cleanup := func() {
		os.Remove(versionsFile)
		os.RemoveAll(filepath.Join(testDir, ".terraform"))
		os.Remove(filepath.Join(testDir, ".terraform.lock.hcl"))
		os.Remove(filepath.Join(testDir, "terraform.tfstate"))
		os.Remove(filepath.Join(testDir, "terraform.tfstate.backup"))
		// Clean up any backup files created by the tool
		backupFiles, _ := filepath.Glob(filepath.Join(testDir, "terraform.tfstate.backup-*"))
		for _, f := range backupFiles {
			os.Remove(f)
		}
	}

	// Clean up before starting
	cleanup()
	defer cleanup()

	// Step 1: Create versions.tf with v3.5.1
	t.Run("setup_v3.5.1", func(t *testing.T) {
		versionsContent := `terraform {
  required_providers {
    random = {
      source  = "hashicorp/random"
      version = "~> 3.5.1"
    }
  }
}
`
		err := os.WriteFile(versionsFile, []byte(versionsContent), 0644)
		if err != nil {
			t.Fatalf("Failed to create versions.tf: %v", err)
		}

		// Run terraform init
		cmd := exec.Command("terraform", "init")
		cmd.Dir = testDir
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("terraform init failed: %v\nOutput: %s", err, output)
		}

		// Run terraform apply
		cmd = exec.Command("terraform", "apply", "-auto-approve")
		cmd.Dir = testDir
		output, err = cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("terraform apply failed: %v\nOutput: %s", err, output)
		}

		t.Logf("Successfully created state with v3.5.1")
	})

	// Step 2: Verify state has resources with schema_version > 0
	t.Run("verify_v3.5.1_state", func(t *testing.T) {
		stateFile := filepath.Join(testDir, "terraform.tfstate")
		data, err := os.ReadFile(stateFile)
		if err != nil {
			t.Fatalf("Failed to read state: %v", err)
		}

		stateContent := string(data)
		if !strings.Contains(stateContent, `"schema_version": 2`) {
			t.Errorf("Expected random_string to have schema_version 2")
		}
		if !strings.Contains(stateContent, `"schema_version": 3`) {
			t.Errorf("Expected random_password to have schema_version 3")
		}

		t.Logf("State contains resources with schema_version 2 and 3")
	})

	// Step 3: Downgrade to v3.1.0 (older version with lower schema versions)
	t.Run("downgrade_to_v3.1.0", func(t *testing.T) {
		versionsContent := `terraform {
  required_providers {
    random = {
      source  = "hashicorp/random"
      version = "~> 3.1.0"
    }
  }
}
`
		err := os.WriteFile(versionsFile, []byte(versionsContent), 0644)
		if err != nil {
			t.Fatalf("Failed to update versions.tf: %v", err)
		}

		// Run terraform init -upgrade
		cmd := exec.Command("terraform", "init", "-upgrade")
		cmd.Dir = testDir
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("terraform init -upgrade failed: %v\nOutput: %s", err, output)
		}

		t.Logf("Successfully downgraded to v3.1.0")
	})

	// Step 4: Run apply command to fix the state
	t.Run("apply_fixes_state", func(t *testing.T) {
		stateFilePath := filepath.Join(testDir, "terraform.tfstate")

		// Capture stdout
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		// Run apply command
		err := cmd.ExecuteWithArgs([]string{
			"apply",
			"--config-dir", testDir,
			"--state-file", stateFilePath,
		})

		// Restore stdout
		w.Close()
		os.Stdout = oldStdout

		var buf bytes.Buffer
		io.Copy(&buf, r)
		outputStr := buf.String()

		t.Logf("Tool output:\n%s", outputStr)

		if err != nil {
			t.Errorf("Apply command failed: %v\nOutput: %s", err, outputStr)
			return
		}

		t.Logf("✓ Apply command completed successfully")
	})

	// Step 5: Verify backup file was created
	t.Run("backup_file_created", func(t *testing.T) {
		backupFiles, err := filepath.Glob(filepath.Join(testDir, "terraform.tfstate.backup-*"))
		if err != nil {
			t.Fatalf("Failed to search for backup files: %v", err)
		}

		if len(backupFiles) == 0 {
			t.Errorf("Expected backup file to be created, but none found")
		} else {
			t.Logf("✓ Backup file created: %s", filepath.Base(backupFiles[0]))
		}
	})

	// Step 6: Verify state schema versions are corrected using strict checking
	t.Run("verify_schema_versions_corrected", func(t *testing.T) {
		stateFile := filepath.Join(testDir, "terraform.tfstate")

		// Parse state to check each resource type
		data, err := os.ReadFile(stateFile)
		if err != nil {
			t.Fatalf("Failed to read state file: %v", err)
		}

		var stateData state.State
		if err := json.Unmarshal(data, &stateData); err != nil {
			t.Fatalf("Failed to parse state JSON: %v", err)
		}

		// Verify each resource has the correct target schema version for provider v3.1.0
		for _, resource := range stateData.Resources {
			if resource.Mode != "managed" {
				continue
			}

			for i, instance := range resource.Instances {
				var expectedVersion int
				switch resource.Type {
				case "random_string":
					expectedVersion = 1 // v3.1.0 target
				case "random_password":
					expectedVersion = 0 // v3.1.0 target
				default:
					t.Errorf("Unexpected resource type: %s", resource.Type)
					continue
				}

				if instance.SchemaVersion != expectedVersion {
					t.Errorf("Resource %s.%s[%d] has schema_version %d, expected %d",
						resource.Type, resource.Name, i, instance.SchemaVersion, expectedVersion)
				}
			}
		}

		t.Logf("✓ All resources have correct schema versions")
	})
}
