// Copyright (c) 2024 Ishaan Kalra
// SPDX-License-Identifier: MIT

package analysis

import (
	"testing"

	"github.com/ishaankalra/terraform-state-downgrader/internal/config"
	"github.com/ishaankalra/terraform-state-downgrader/internal/state"
)

func TestDetectMismatches_DowngradeNeeded(t *testing.T) {
	// Mock state with schema version 1
	stateData := &state.State{
		Resources: []state.Resource{
			{
				Mode: "managed",
				Type: "aws_instance",
				Name: "web",
				Instances: []state.ResourceInstance{
					{
						SchemaVersion: 1,
						Attributes: map[string]interface{}{
							"id":            "i-1234567890abcdef0",
							"instance_type": "t2.micro",
						},
					},
				},
			},
		},
	}

	// Mock resource mapping from terraform show
	resourceMapping := map[string]string{
		"aws_instance.web": "registry.terraform.io/hashicorp/aws",
	}

	// Mock schema versions from provider (target version is 0)
	schemaVersions := map[string]map[string]int64{
		"registry.terraform.io/hashicorp/aws": {
			"aws_instance": 0,
		},
	}

	// Detect mismatches
	fullSchemas := make(map[string]map[string]config.SchemaDefinition)
	mismatches, err := DetectMismatches(stateData, resourceMapping, schemaVersions, fullSchemas)
	if err != nil {
		t.Fatalf("DetectMismatches failed: %v", err)
	}

	// Assertions
	if len(mismatches) != 1 {
		t.Fatalf("Expected 1 mismatch, got %d", len(mismatches))
	}

	mismatch := mismatches[0]
	if mismatch.ResourceAddress != "aws_instance.web" {
		t.Errorf("Expected address aws_instance.web, got %s", mismatch.ResourceAddress)
	}

	if mismatch.StateVersion != 1 {
		t.Errorf("Expected state version 1, got %d", mismatch.StateVersion)
	}

	if mismatch.TargetVersion != 0 {
		t.Errorf("Expected target version 0, got %d", mismatch.TargetVersion)
	}

	if mismatch.ResourceID != "i-1234567890abcdef0" {
		t.Errorf("Expected resource ID i-1234567890abcdef0, got %s", mismatch.ResourceID)
	}
}

func TestDetectMismatches_NoMismatch(t *testing.T) {
	// State and schema have same version
	stateData := &state.State{
		Resources: []state.Resource{
			{
				Mode: "managed",
				Type: "aws_instance",
				Name: "web",
				Instances: []state.ResourceInstance{
					{
						SchemaVersion: 0,
						Attributes: map[string]interface{}{
							"id": "i-12345",
						},
					},
				},
			},
		},
	}

	resourceMapping := map[string]string{
		"aws_instance.web": "registry.terraform.io/hashicorp/aws",
	}

	schemaVersions := map[string]map[string]int64{
		"registry.terraform.io/hashicorp/aws": {
			"aws_instance": 0,
		},
	}

	fullSchemas := make(map[string]map[string]config.SchemaDefinition)
	mismatches, err := DetectMismatches(stateData, resourceMapping, schemaVersions, fullSchemas)
	if err != nil {
		t.Fatalf("DetectMismatches failed: %v", err)
	}

	if len(mismatches) != 0 {
		t.Fatalf("Expected 0 mismatches, got %d", len(mismatches))
	}
}

func TestDetectMismatches_MultipleResources(t *testing.T) {
	stateData := &state.State{
		Resources: []state.Resource{
			{
				Mode: "managed",
				Type: "aws_instance",
				Name: "web",
				Instances: []state.ResourceInstance{
					{
						SchemaVersion: 1,
						Attributes:    map[string]interface{}{"id": "i-111"},
					},
				},
			},
			{
				Mode: "managed",
				Type: "aws_db_instance",
				Name: "database",
				Instances: []state.ResourceInstance{
					{
						SchemaVersion: 2,
						Attributes:    map[string]interface{}{"id": "db-222"},
					},
				},
			},
			{
				Mode: "managed",
				Type: "aws_s3_bucket",
				Name: "storage",
				Instances: []state.ResourceInstance{
					{
						SchemaVersion: 0,
						Attributes:    map[string]interface{}{"id": "bucket-333"},
					},
				},
			},
		},
	}

	resourceMapping := map[string]string{
		"aws_instance.web":      "registry.terraform.io/hashicorp/aws",
		"aws_db_instance.database": "registry.terraform.io/hashicorp/aws",
		"aws_s3_bucket.storage": "registry.terraform.io/hashicorp/aws",
	}

	schemaVersions := map[string]map[string]int64{
		"registry.terraform.io/hashicorp/aws": {
			"aws_instance":    0,
			"aws_db_instance": 1, // Only db downgrade needed (2 → 1)
			"aws_s3_bucket":   0,
		},
	}

	fullSchemas := make(map[string]map[string]config.SchemaDefinition)
	mismatches, err := DetectMismatches(stateData, resourceMapping, schemaVersions, fullSchemas)
	if err != nil {
		t.Fatalf("DetectMismatches failed: %v", err)
	}

	// Should detect 2 mismatches: aws_instance (1→0) and aws_db_instance (2→1)
	if len(mismatches) != 2 {
		t.Fatalf("Expected 2 mismatches, got %d", len(mismatches))
	}

	// Check that we got the right resources
	foundInstance := false
	foundDB := false
	for _, m := range mismatches {
		if m.ResourceType == "aws_instance" && m.StateVersion == 1 && m.TargetVersion == 0 {
			foundInstance = true
		}
		if m.ResourceType == "aws_db_instance" && m.StateVersion == 2 && m.TargetVersion == 1 {
			foundDB = true
		}
	}

	if !foundInstance {
		t.Error("Expected to find aws_instance mismatch")
	}
	if !foundDB {
		t.Error("Expected to find aws_db_instance mismatch")
	}
}

func TestDetectMismatches_ResourceWithCount(t *testing.T) {
	stateData := &state.State{
		Resources: []state.Resource{
			{
				Mode: "managed",
				Type: "aws_instance",
				Name: "web",
				Instances: []state.ResourceInstance{
					{
						IndexKey:      float64(0),
						SchemaVersion: 0,
						Attributes:    map[string]interface{}{"id": "i-000"},
					},
					{
						IndexKey:      float64(1),
						SchemaVersion: 1, // Only this one needs downgrade
						Attributes:    map[string]interface{}{"id": "i-001"},
					},
					{
						IndexKey:      float64(2),
						SchemaVersion: 0,
						Attributes:    map[string]interface{}{"id": "i-002"},
					},
				},
			},
		},
	}

	resourceMapping := map[string]string{
		"aws_instance.web": "registry.terraform.io/hashicorp/aws",
	}

	schemaVersions := map[string]map[string]int64{
		"registry.terraform.io/hashicorp/aws": {
			"aws_instance": 0,
		},
	}

	fullSchemas := make(map[string]map[string]config.SchemaDefinition)
	mismatches, err := DetectMismatches(stateData, resourceMapping, schemaVersions, fullSchemas)
	if err != nil {
		t.Fatalf("DetectMismatches failed: %v", err)
	}

	// Should only detect instance[1]
	if len(mismatches) != 1 {
		t.Fatalf("Expected 1 mismatch, got %d", len(mismatches))
	}

	mismatch := mismatches[0]
	if mismatch.InstanceIndex != 1 {
		t.Errorf("Expected instance index 1, got %d", mismatch.InstanceIndex)
	}

	if mismatch.ResourceID != "i-001" {
		t.Errorf("Expected resource ID i-001, got %s", mismatch.ResourceID)
	}

	// Verify address includes count index
	if mismatch.ResourceAddress != "aws_instance.web[1]" {
		t.Errorf("Expected resource address aws_instance.web[1], got %s", mismatch.ResourceAddress)
	}
}

func TestDetectMismatches_WithTimeouts(t *testing.T) {
	stateData := &state.State{
		Resources: []state.Resource{
			{
				Mode: "managed",
				Type: "aws_db_instance",
				Name: "main",
				Instances: []state.ResourceInstance{
					{
						SchemaVersion: 2,
						Attributes: map[string]interface{}{
							"id": "db-12345",
							"timeouts": map[string]interface{}{
								"create": "60m",
								"update": "80m",
								"delete": "60m",
							},
						},
					},
				},
			},
		},
	}

	resourceMapping := map[string]string{
		"aws_db_instance.main": "registry.terraform.io/hashicorp/aws",
	}

	schemaVersions := map[string]map[string]int64{
		"registry.terraform.io/hashicorp/aws": {
			"aws_db_instance": 1,
		},
	}

	fullSchemas := make(map[string]map[string]config.SchemaDefinition)
	mismatches, err := DetectMismatches(stateData, resourceMapping, schemaVersions, fullSchemas)
	if err != nil {
		t.Fatalf("DetectMismatches failed: %v", err)
	}

	if len(mismatches) != 1 {
		t.Fatalf("Expected 1 mismatch, got %d", len(mismatches))
	}

	mismatch := mismatches[0]
	if mismatch.Timeouts == nil {
		t.Fatal("Expected timeouts to be extracted, got nil")
	}

	if create, ok := mismatch.Timeouts["create"].(string); !ok || create != "60m" {
		t.Errorf("Expected timeout create=60m, got %v", mismatch.Timeouts["create"])
	}
}

func TestDetectMismatches_SkipDataSources(t *testing.T) {
	stateData := &state.State{
		Resources: []state.Resource{
			{
				Mode: "data",
				Type: "aws_ami",
				Name: "ubuntu",
				Instances: []state.ResourceInstance{
					{
						SchemaVersion: 1,
						Attributes:    map[string]interface{}{"id": "ami-12345"},
					},
				},
			},
		},
	}

	resourceMapping := map[string]string{
		"aws_ami.ubuntu": "registry.terraform.io/hashicorp/aws",
	}

	schemaVersions := map[string]map[string]int64{
		"registry.terraform.io/hashicorp/aws": {
			"aws_ami": 0,
		},
	}

	fullSchemas := make(map[string]map[string]config.SchemaDefinition)
	mismatches, err := DetectMismatches(stateData, resourceMapping, schemaVersions, fullSchemas)
	if err != nil {
		t.Fatalf("DetectMismatches failed: %v", err)
	}

	// Data sources should be skipped
	if len(mismatches) != 0 {
		t.Fatalf("Expected 0 mismatches (data sources should be skipped), got %d", len(mismatches))
	}
}

func TestDetectMismatches_ResourceNotInConfig(t *testing.T) {
	stateData := &state.State{
		Resources: []state.Resource{
			{
				Mode: "managed",
				Type: "aws_instance",
				Name: "deleted",
				Instances: []state.ResourceInstance{
					{
						SchemaVersion: 1,
						Attributes:    map[string]interface{}{"id": "i-deleted"},
					},
				},
			},
		},
	}

	// Resource not in mapping (removed from config)
	resourceMapping := map[string]string{}

	schemaVersions := map[string]map[string]int64{
		"registry.terraform.io/hashicorp/aws": {
			"aws_instance": 0,
		},
	}

	fullSchemas := make(map[string]map[string]config.SchemaDefinition)
	mismatches, err := DetectMismatches(stateData, resourceMapping, schemaVersions, fullSchemas)
	if err != nil {
		t.Fatalf("DetectMismatches failed: %v", err)
	}

	// Should skip resource not in config
	if len(mismatches) != 0 {
		t.Fatalf("Expected 0 mismatches (resource not in config), got %d", len(mismatches))
	}
}

func TestDetectMismatches_ProviderNotFound(t *testing.T) {
	stateData := &state.State{
		Resources: []state.Resource{
			{
				Mode: "managed",
				Type: "aws_instance",
				Name: "web",
				Instances: []state.ResourceInstance{
					{
						SchemaVersion: 1,
						Attributes:    map[string]interface{}{"id": "i-12345"},
					},
				},
			},
		},
	}

	resourceMapping := map[string]string{
		"aws_instance.web": "registry.terraform.io/hashicorp/aws",
	}

	// Provider not in schema versions
	schemaVersions := map[string]map[string]int64{}

	fullSchemas := make(map[string]map[string]config.SchemaDefinition)
	mismatches, err := DetectMismatches(stateData, resourceMapping, schemaVersions, fullSchemas)
	if err != nil {
		t.Fatalf("DetectMismatches failed: %v", err)
	}

	// Should skip when provider not found
	if len(mismatches) != 0 {
		t.Fatalf("Expected 0 mismatches (provider not found), got %d", len(mismatches))
	}
}

func TestDetectMismatches_ResourceTypeNotInSchema(t *testing.T) {
	stateData := &state.State{
		Resources: []state.Resource{
			{
				Mode: "managed",
				Type: "aws_new_resource",
				Name: "test",
				Instances: []state.ResourceInstance{
					{
						SchemaVersion: 1,
						Attributes:    map[string]interface{}{"id": "new-123"},
					},
				},
			},
		},
	}

	resourceMapping := map[string]string{
		"aws_new_resource.test": "registry.terraform.io/hashicorp/aws",
	}

	schemaVersions := map[string]map[string]int64{
		"registry.terraform.io/hashicorp/aws": {
			"aws_instance": 0,
			// aws_new_resource not in schema
		},
	}

	fullSchemas := make(map[string]map[string]config.SchemaDefinition)
	mismatches, err := DetectMismatches(stateData, resourceMapping, schemaVersions, fullSchemas)
	if err != nil {
		t.Fatalf("DetectMismatches failed: %v", err)
	}

	// Should skip when resource type not in schema
	if len(mismatches) != 0 {
		t.Fatalf("Expected 0 mismatches (resource type not in schema), got %d", len(mismatches))
	}
}

func TestDetectMismatches_MultipleProviders(t *testing.T) {
	stateData := &state.State{
		Resources: []state.Resource{
			{
				Mode: "managed",
				Type: "aws_instance",
				Name: "web",
				Instances: []state.ResourceInstance{
					{
						SchemaVersion: 1,
						Attributes:    map[string]interface{}{"id": "i-aws"},
					},
				},
			},
			{
				Mode: "managed",
				Type: "random_id",
				Name: "suffix",
				Instances: []state.ResourceInstance{
					{
						SchemaVersion: 1,
						Attributes:    map[string]interface{}{"id": "random-123"},
					},
				},
			},
		},
	}

	resourceMapping := map[string]string{
		"aws_instance.web": "registry.terraform.io/hashicorp/aws",
		"random_id.suffix": "registry.terraform.io/hashicorp/random",
	}

	schemaVersions := map[string]map[string]int64{
		"registry.terraform.io/hashicorp/aws": {
			"aws_instance": 0,
		},
		"registry.terraform.io/hashicorp/random": {
			"random_id": 0,
		},
	}

	fullSchemas := make(map[string]map[string]config.SchemaDefinition)
	mismatches, err := DetectMismatches(stateData, resourceMapping, schemaVersions, fullSchemas)
	if err != nil {
		t.Fatalf("DetectMismatches failed: %v", err)
	}

	// Both resources need downgrade
	if len(mismatches) != 2 {
		t.Fatalf("Expected 2 mismatches, got %d", len(mismatches))
	}

	// Verify providers are correct
	foundAWS := false
	foundRandom := false
	for _, m := range mismatches {
		if m.ProviderAddress == "registry.terraform.io/hashicorp/aws" {
			foundAWS = true
		}
		if m.ProviderAddress == "registry.terraform.io/hashicorp/random" {
			foundRandom = true
		}
	}

	if !foundAWS || !foundRandom {
		t.Error("Expected mismatches from both AWS and Random providers")
	}
}

func TestDetectMismatches_ResourceWithForEach(t *testing.T) {
	// Mock state with resource using for_each
	stateData := &state.State{
		Resources: []state.Resource{
			{
				Mode: "managed",
				Type: "aws_s3_bucket",
				Name: "example",
				Instances: []state.ResourceInstance{
					{
						SchemaVersion: 2,
						IndexKey:      "prod", // for_each key
						Attributes: map[string]interface{}{
							"id": "prod-bucket",
						},
					},
					{
						SchemaVersion: 2,
						IndexKey:      "staging", // for_each key
						Attributes: map[string]interface{}{
							"id": "staging-bucket",
						},
					},
				},
			},
		},
	}

	resourceMapping := map[string]string{
		`aws_s3_bucket.example["prod"]`:    "registry.terraform.io/hashicorp/aws",
		`aws_s3_bucket.example["staging"]`: "registry.terraform.io/hashicorp/aws",
	}

	schemaVersions := map[string]map[string]int64{
		"registry.terraform.io/hashicorp/aws": {
			"aws_s3_bucket": 0,
		},
	}

	fullSchemas := make(map[string]map[string]config.SchemaDefinition)
	mismatches, err := DetectMismatches(stateData, resourceMapping, schemaVersions, fullSchemas)
	if err != nil {
		t.Fatalf("DetectMismatches failed: %v", err)
	}

	// Should detect 2 mismatches (one per instance)
	if len(mismatches) != 2 {
		t.Fatalf("Expected 2 mismatches, got %d", len(mismatches))
	}

	// Verify addresses include for_each keys
	expectedAddresses := map[string]bool{
		`aws_s3_bucket.example["prod"]`:    false,
		`aws_s3_bucket.example["staging"]`: false,
	}

	for _, m := range mismatches {
		if _, exists := expectedAddresses[m.ResourceAddress]; exists {
			expectedAddresses[m.ResourceAddress] = true
		} else {
			t.Errorf("Unexpected resource address: %s", m.ResourceAddress)
		}
	}

	for addr, found := range expectedAddresses {
		if !found {
			t.Errorf("Missing expected resource address: %s", addr)
		}
	}
}

func TestDetectMismatches_SchemaValidation_MissingAttribute(t *testing.T) {
	// State has an attribute that doesn't exist in provider schema
	stateData := &state.State{
		Resources: []state.Resource{
			{
				Mode: "managed",
				Type: "aws_instance",
				Name: "web",
				Instances: []state.ResourceInstance{
					{
						SchemaVersion: 0, // Version matches
						Attributes: map[string]interface{}{
							"id":               "i-12345",
							"instance_type":    "t2.micro",
							"deprecated_field": "old_value", // This field doesn't exist in schema
						},
					},
				},
			},
		},
	}

	resourceMapping := map[string]string{
		"aws_instance.web": "registry.terraform.io/hashicorp/aws",
	}

	schemaVersions := map[string]map[string]int64{
		"registry.terraform.io/hashicorp/aws": {
			"aws_instance": 0,
		},
	}

	// Full schema with Block structure
	fullSchemas := map[string]map[string]config.SchemaDefinition{
		"registry.terraform.io/hashicorp/aws": {
			"aws_instance": {
				Version: 0,
				Block: map[string]interface{}{
					"attributes": map[string]interface{}{
						"id": map[string]interface{}{
							"type":     "string",
							"computed": true,
						},
						"instance_type": map[string]interface{}{
							"type":     "string",
							"required": true,
						},
						// Note: deprecated_field is NOT in the schema
					},
				},
			},
		},
	}

	mismatches, err := DetectMismatches(stateData, resourceMapping, schemaVersions, fullSchemas)
	if err != nil {
		t.Fatalf("DetectMismatches failed: %v", err)
	}

	// Should detect schema issue (missing attribute in schema)
	if len(mismatches) != 1 {
		t.Fatalf("Expected 1 mismatch, got %d", len(mismatches))
	}

	mismatch := mismatches[0]
	if !mismatch.HasSchemaIssues {
		t.Error("Expected HasSchemaIssues to be true")
	}

	if mismatch.HasVersionMismatch {
		t.Error("Expected HasVersionMismatch to be false (versions match)")
	}

	if len(mismatch.SchemaIssues) == 0 {
		t.Fatal("Expected schema issues to be populated")
	}

	// Check that the issue mentions the deprecated field
	foundDeprecatedIssue := false
	for _, issue := range mismatch.SchemaIssues {
		if len(issue) > 0 && (issue == "attribute 'deprecated_field' not found in provider schema") {
			foundDeprecatedIssue = true
		}
	}

	if !foundDeprecatedIssue {
		t.Errorf("Expected to find deprecated_field issue, got: %v", mismatch.SchemaIssues)
	}
}

func TestDetectMismatches_SchemaValidation_MissingRequiredAttribute(t *testing.T) {
	// State is missing a required attribute
	stateData := &state.State{
		Resources: []state.Resource{
			{
				Mode: "managed",
				Type: "aws_s3_bucket",
				Name: "data",
				Instances: []state.ResourceInstance{
					{
						SchemaVersion: 0,
						Attributes: map[string]interface{}{
							"id": "my-bucket",
							// Missing 'bucket' which is required
						},
					},
				},
			},
		},
	}

	resourceMapping := map[string]string{
		"aws_s3_bucket.data": "registry.terraform.io/hashicorp/aws",
	}

	schemaVersions := map[string]map[string]int64{
		"registry.terraform.io/hashicorp/aws": {
			"aws_s3_bucket": 0,
		},
	}

	fullSchemas := map[string]map[string]config.SchemaDefinition{
		"registry.terraform.io/hashicorp/aws": {
			"aws_s3_bucket": {
				Version: 0,
				Block: map[string]interface{}{
					"attributes": map[string]interface{}{
						"id": map[string]interface{}{
							"type":     "string",
							"computed": true,
						},
						"bucket": map[string]interface{}{
							"type":     "string",
							"required": true, // This is required but missing in state
						},
						"region": map[string]interface{}{
							"type":     "string",
							"optional": true,
						},
					},
				},
			},
		},
	}

	mismatches, err := DetectMismatches(stateData, resourceMapping, schemaVersions, fullSchemas)
	if err != nil {
		t.Fatalf("DetectMismatches failed: %v", err)
	}

	if len(mismatches) != 1 {
		t.Fatalf("Expected 1 mismatch, got %d", len(mismatches))
	}

	mismatch := mismatches[0]
	if !mismatch.HasSchemaIssues {
		t.Error("Expected HasSchemaIssues to be true")
	}

	// Check for required attribute issue
	foundRequiredIssue := false
	for _, issue := range mismatch.SchemaIssues {
		if len(issue) > 0 && (issue == "required attribute 'bucket' missing from state") {
			foundRequiredIssue = true
		}
	}

	if !foundRequiredIssue {
		t.Errorf("Expected to find missing required 'bucket' issue, got: %v", mismatch.SchemaIssues)
	}
}

func TestDetectMismatches_SchemaValidation_TypeMismatch(t *testing.T) {
	// State has wrong type for an attribute
	stateData := &state.State{
		Resources: []state.Resource{
			{
				Mode: "managed",
				Type: "aws_instance",
				Name: "web",
				Instances: []state.ResourceInstance{
					{
						SchemaVersion: 0,
						Attributes: map[string]interface{}{
							"id":            "i-12345",
							"instance_type": 12345, // Should be string, not number
						},
					},
				},
			},
		},
	}

	resourceMapping := map[string]string{
		"aws_instance.web": "registry.terraform.io/hashicorp/aws",
	}

	schemaVersions := map[string]map[string]int64{
		"registry.terraform.io/hashicorp/aws": {
			"aws_instance": 0,
		},
	}

	fullSchemas := map[string]map[string]config.SchemaDefinition{
		"registry.terraform.io/hashicorp/aws": {
			"aws_instance": {
				Version: 0,
				Block: map[string]interface{}{
					"attributes": map[string]interface{}{
						"id": map[string]interface{}{
							"type": "string",
						},
						"instance_type": map[string]interface{}{
							"type":     "string", // Expects string
							"required": true,
						},
					},
				},
			},
		},
	}

	mismatches, err := DetectMismatches(stateData, resourceMapping, schemaVersions, fullSchemas)
	if err != nil {
		t.Fatalf("DetectMismatches failed: %v", err)
	}

	if len(mismatches) != 1 {
		t.Fatalf("Expected 1 mismatch, got %d", len(mismatches))
	}

	mismatch := mismatches[0]
	if !mismatch.HasSchemaIssues {
		t.Error("Expected HasSchemaIssues to be true")
	}

	// Check for type mismatch
	foundTypeIssue := false
	for _, issue := range mismatch.SchemaIssues {
		if len(issue) > 0 && (issue == "attribute 'instance_type' should be string, got int") {
			foundTypeIssue = true
		}
	}

	if !foundTypeIssue {
		t.Errorf("Expected to find type mismatch issue for instance_type, got: %v", mismatch.SchemaIssues)
	}
}

func TestDetectMismatches_SchemaValidation_ValidSchema(t *testing.T) {
	// State matches schema perfectly - no issues
	stateData := &state.State{
		Resources: []state.Resource{
			{
				Mode: "managed",
				Type: "aws_instance",
				Name: "web",
				Instances: []state.ResourceInstance{
					{
						SchemaVersion: 0,
						Attributes: map[string]interface{}{
							"id":            "i-12345",
							"instance_type": "t2.micro",
							"ami":           "ami-12345",
						},
					},
				},
			},
		},
	}

	resourceMapping := map[string]string{
		"aws_instance.web": "registry.terraform.io/hashicorp/aws",
	}

	schemaVersions := map[string]map[string]int64{
		"registry.terraform.io/hashicorp/aws": {
			"aws_instance": 0,
		},
	}

	fullSchemas := map[string]map[string]config.SchemaDefinition{
		"registry.terraform.io/hashicorp/aws": {
			"aws_instance": {
				Version: 0,
				Block: map[string]interface{}{
					"attributes": map[string]interface{}{
						"id": map[string]interface{}{
							"type":     "string",
							"computed": true,
						},
						"instance_type": map[string]interface{}{
							"type":     "string",
							"required": true,
						},
						"ami": map[string]interface{}{
							"type":     "string",
							"required": true,
						},
					},
				},
			},
		},
	}

	mismatches, err := DetectMismatches(stateData, resourceMapping, schemaVersions, fullSchemas)
	if err != nil {
		t.Fatalf("DetectMismatches failed: %v", err)
	}

	// Should have NO mismatches - everything is valid
	if len(mismatches) != 0 {
		t.Fatalf("Expected 0 mismatches (all valid), got %d: %v", len(mismatches), mismatches)
	}
}

func TestDetectMismatches_SchemaValidation_VersionMismatchAndSchemaIssues(t *testing.T) {
	// Resource has BOTH version mismatch AND schema issues
	stateData := &state.State{
		Resources: []state.Resource{
			{
				Mode: "managed",
				Type: "aws_db_instance",
				Name: "main",
				Instances: []state.ResourceInstance{
					{
						SchemaVersion: 2, // Wrong version (should be 1)
						Attributes: map[string]interface{}{
							"id":               "db-12345",
							"instance_class":   "db.t2.micro",
							"deprecated_field": "old_value", // Not in schema
						},
					},
				},
			},
		},
	}

	resourceMapping := map[string]string{
		"aws_db_instance.main": "registry.terraform.io/hashicorp/aws",
	}

	schemaVersions := map[string]map[string]int64{
		"registry.terraform.io/hashicorp/aws": {
			"aws_db_instance": 1, // Target version is 1, state has 2
		},
	}

	fullSchemas := map[string]map[string]config.SchemaDefinition{
		"registry.terraform.io/hashicorp/aws": {
			"aws_db_instance": {
				Version: 1,
				Block: map[string]interface{}{
					"attributes": map[string]interface{}{
						"id": map[string]interface{}{
							"type":     "string",
							"computed": true,
						},
						"instance_class": map[string]interface{}{
							"type":     "string",
							"required": true,
						},
						"engine": map[string]interface{}{
							"type":     "string",
							"required": true, // Missing in state
						},
						// deprecated_field is NOT in schema
					},
				},
			},
		},
	}

	mismatches, err := DetectMismatches(stateData, resourceMapping, schemaVersions, fullSchemas)
	if err != nil {
		t.Fatalf("DetectMismatches failed: %v", err)
	}

	if len(mismatches) != 1 {
		t.Fatalf("Expected 1 mismatch, got %d", len(mismatches))
	}

	mismatch := mismatches[0]

	// Should have BOTH version mismatch AND schema issues
	if !mismatch.HasVersionMismatch {
		t.Error("Expected HasVersionMismatch to be true")
	}

	if !mismatch.HasSchemaIssues {
		t.Error("Expected HasSchemaIssues to be true")
	}

	if mismatch.StateVersion != 2 {
		t.Errorf("Expected state version 2, got %d", mismatch.StateVersion)
	}

	if mismatch.TargetVersion != 1 {
		t.Errorf("Expected target version 1, got %d", mismatch.TargetVersion)
	}

	// Should have at least 2 schema issues
	if len(mismatch.SchemaIssues) < 2 {
		t.Errorf("Expected at least 2 schema issues, got %d: %v", len(mismatch.SchemaIssues), mismatch.SchemaIssues)
	}

	// Verify both issues are detected
	foundDeprecatedIssue := false
	foundMissingEngineIssue := false
	for _, issue := range mismatch.SchemaIssues {
		if issue == "attribute 'deprecated_field' not found in provider schema" {
			foundDeprecatedIssue = true
		}
		if issue == "required attribute 'engine' missing from state" {
			foundMissingEngineIssue = true
		}
	}

	if !foundDeprecatedIssue {
		t.Error("Expected to find deprecated_field issue")
	}
	if !foundMissingEngineIssue {
		t.Error("Expected to find missing 'engine' issue")
	}
}
