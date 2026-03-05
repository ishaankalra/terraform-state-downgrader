// Copyright (c) 2024 Ishaan Kalra
// SPDX-License-Identifier: MIT

package state

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadState_ValidState(t *testing.T) {
	// Create a temporary state file with mock data
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "terraform.tfstate")

	mockState := `{
  "version": 4,
  "terraform_version": "1.5.0",
  "serial": 1,
  "lineage": "test-lineage",
  "resources": [
    {
      "mode": "managed",
      "type": "aws_instance",
      "name": "web",
      "provider": "provider[\"registry.terraform.io/hashicorp/aws\"]",
      "instances": [
        {
          "schema_version": 1,
          "attributes": {
            "id": "i-1234567890abcdef0",
            "ami": "ami-12345678",
            "instance_type": "t2.micro"
          }
        }
      ]
    }
  ]
}`

	err := os.WriteFile(statePath, []byte(mockState), 0644)
	if err != nil {
		t.Fatalf("Failed to create temp state file: %v", err)
	}

	// Test reading the state
	state, err := ReadState(statePath)
	if err != nil {
		t.Fatalf("ReadState failed: %v", err)
	}

	// Assertions
	if state.Version != 4 {
		t.Errorf("Expected version 4, got %d", state.Version)
	}

	if state.TerraformVersion != "1.5.0" {
		t.Errorf("Expected terraform_version 1.5.0, got %s", state.TerraformVersion)
	}

	if len(state.Resources) != 1 {
		t.Fatalf("Expected 1 resource, got %d", len(state.Resources))
	}

	resource := state.Resources[0]
	if resource.Type != "aws_instance" {
		t.Errorf("Expected type aws_instance, got %s", resource.Type)
	}

	if resource.Name != "web" {
		t.Errorf("Expected name web, got %s", resource.Name)
	}

	if len(resource.Instances) != 1 {
		t.Fatalf("Expected 1 instance, got %d", len(resource.Instances))
	}

	instance := resource.Instances[0]
	if instance.SchemaVersion != 1 {
		t.Errorf("Expected schema_version 1, got %d", instance.SchemaVersion)
	}

	if id, ok := instance.Attributes["id"].(string); !ok || id != "i-1234567890abcdef0" {
		t.Errorf("Expected id i-1234567890abcdef0, got %v", instance.Attributes["id"])
	}
}

func TestReadState_MultipleResources(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "terraform.tfstate")

	mockState := `{
  "version": 4,
  "terraform_version": "1.5.0",
  "resources": [
    {
      "mode": "managed",
      "type": "aws_instance",
      "name": "web",
      "provider": "provider[\"registry.terraform.io/hashicorp/aws\"]",
      "instances": [
        {
          "schema_version": 1,
          "attributes": {"id": "i-111"}
        }
      ]
    },
    {
      "mode": "managed",
      "type": "aws_db_instance",
      "name": "database",
      "provider": "provider[\"registry.terraform.io/hashicorp/aws\"]",
      "instances": [
        {
          "schema_version": 2,
          "attributes": {"id": "db-222"}
        }
      ]
    }
  ]
}`

	os.WriteFile(statePath, []byte(mockState), 0644)

	state, err := ReadState(statePath)
	if err != nil {
		t.Fatalf("ReadState failed: %v", err)
	}

	if len(state.Resources) != 2 {
		t.Fatalf("Expected 2 resources, got %d", len(state.Resources))
	}

	// Verify first resource
	if state.Resources[0].Type != "aws_instance" {
		t.Errorf("Expected first resource type aws_instance, got %s", state.Resources[0].Type)
	}

	// Verify second resource
	if state.Resources[1].Type != "aws_db_instance" {
		t.Errorf("Expected second resource type aws_db_instance, got %s", state.Resources[1].Type)
	}
}

func TestReadState_ResourceWithCount(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "terraform.tfstate")

	mockState := `{
  "version": 4,
  "terraform_version": "1.5.0",
  "resources": [
    {
      "mode": "managed",
      "type": "aws_instance",
      "name": "web",
      "provider": "provider[\"registry.terraform.io/hashicorp/aws\"]",
      "instances": [
        {
          "index_key": 0,
          "schema_version": 1,
          "attributes": {"id": "i-000"}
        },
        {
          "index_key": 1,
          "schema_version": 1,
          "attributes": {"id": "i-001"}
        },
        {
          "index_key": 2,
          "schema_version": 1,
          "attributes": {"id": "i-002"}
        }
      ]
    }
  ]
}`

	os.WriteFile(statePath, []byte(mockState), 0644)

	state, err := ReadState(statePath)
	if err != nil {
		t.Fatalf("ReadState failed: %v", err)
	}

	if len(state.Resources) != 1 {
		t.Fatalf("Expected 1 resource, got %d", len(state.Resources))
	}

	resource := state.Resources[0]
	if len(resource.Instances) != 3 {
		t.Fatalf("Expected 3 instances, got %d", len(resource.Instances))
	}

	// Verify instance indices (JSON unmarshals numbers as float64)
	for i, instance := range resource.Instances {
		expectedKey := float64(i)
		if instance.IndexKey != expectedKey {
			t.Errorf("Expected index_key %v, got %v", expectedKey, instance.IndexKey)
		}
	}
}

func TestReadState_FileNotFound(t *testing.T) {
	_, err := ReadState("/nonexistent/path/terraform.tfstate")
	if err == nil {
		t.Fatal("Expected error for nonexistent file, got nil")
	}
}

func TestReadState_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "terraform.tfstate")

	// Write invalid JSON
	invalidJSON := `{this is not valid json`
	os.WriteFile(statePath, []byte(invalidJSON), 0644)

	_, err := ReadState(statePath)
	if err == nil {
		t.Fatal("Expected error for invalid JSON, got nil")
	}
}

func TestReadState_EmptyState(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "terraform.tfstate")

	emptyState := `{
  "version": 4,
  "terraform_version": "1.5.0",
  "resources": []
}`

	os.WriteFile(statePath, []byte(emptyState), 0644)

	state, err := ReadState(statePath)
	if err != nil {
		t.Fatalf("ReadState failed: %v", err)
	}

	if len(state.Resources) != 0 {
		t.Errorf("Expected 0 resources, got %d", len(state.Resources))
	}
}

func TestReadState_DataSource(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "terraform.tfstate")

	mockState := `{
  "version": 4,
  "terraform_version": "1.5.0",
  "resources": [
    {
      "mode": "data",
      "type": "aws_ami",
      "name": "ubuntu",
      "provider": "provider[\"registry.terraform.io/hashicorp/aws\"]",
      "instances": [
        {
          "schema_version": 0,
          "attributes": {"id": "ami-12345"}
        }
      ]
    }
  ]
}`

	os.WriteFile(statePath, []byte(mockState), 0644)

	state, err := ReadState(statePath)
	if err != nil {
		t.Fatalf("ReadState failed: %v", err)
	}

	if len(state.Resources) != 1 {
		t.Fatalf("Expected 1 resource, got %d", len(state.Resources))
	}

	if state.Resources[0].Mode != "data" {
		t.Errorf("Expected mode 'data', got %s", state.Resources[0].Mode)
	}
}
