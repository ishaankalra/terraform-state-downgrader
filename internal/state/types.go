// Copyright (c) 2024 Ishaan Kalra
// SPDX-License-Identifier: MIT

package state

// State represents a Terraform state file
type State struct {
	Version          int                `json:"version"`
	TerraformVersion string             `json:"terraform_version"`
	Serial           int                `json:"serial"`
	Lineage          string             `json:"lineage"`
	Outputs          map[string]Output  `json:"outputs"`
	Resources        []Resource         `json:"resources"`
}

// Output represents a Terraform output value
type Output struct {
	Value     interface{} `json:"value"`
	Type      interface{} `json:"type"`
	Sensitive bool        `json:"sensitive"`
}

// Resource represents a resource in the state
type Resource struct {
	Mode         string             `json:"mode"`
	Type         string             `json:"type"`
	Name         string             `json:"name"`
	Provider     string             `json:"provider"`
	Instances    []ResourceInstance `json:"instances"`
	EachMode     *string            `json:"each,omitempty"`
}

// ResourceInstance represents an instance of a resource
type ResourceInstance struct {
	SchemaVersion    int                    `json:"schema_version"`
	Attributes       map[string]interface{} `json:"attributes"`
	Private          string                 `json:"private,omitempty"`
	Dependencies     []string               `json:"dependencies,omitempty"`
	IndexKey         interface{}            `json:"index_key,omitempty"`
	Status           *string                `json:"status,omitempty"`
}