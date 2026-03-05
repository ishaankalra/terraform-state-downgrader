// Copyright (c) 2024 Ishaan Kalra
// SPDX-License-Identifier: MIT

package analysis

// Mismatch represents a resource that needs schema version downgrade
type Mismatch struct {
	ResourceAddress string // e.g. "aws_instance.web"
	ResourceType    string // e.g. "aws_instance"
	ResourceName    string // e.g. "web"
	ProviderAddress string // e.g. "registry.terraform.io/hashicorp/aws"
	StateVersion    int64  // Current schema version in state
	TargetVersion   int64  // Target schema version from provider
	ResourceID      string // Resource ID (e.g. "i-1234567")
	Timeouts        map[string]interface{} // Timeout configuration
	InstanceIndex   int    // Index in resource instances array
}