# terraform-state-downgrader

A CLI tool that helps you downgrade Terraform state schema versions when you switch to an older provider version.

## Problem

When you accidentally upgrade a Terraform provider (e.g., AWS provider v3 → v6) and apply changes, your state file gets upgraded to a newer schema version. If you later try to downgrade back to the older provider version, Terraform will fail with:

```
Error: Resource instance managed by newer provider version
The current state of <resource> was created by a newer provider version
than is currently selected. Upgrade the provider to work with this state.
```

This tool solves that problem by automatically re-importing resources from your cloud provider with the correct schema version.

## How It Works

1. **Analyzes** your configuration using `terraform providers` (parses provider tree) and reads your state file directly
2. **Gets schema versions** using `terraform providers schema -json`
3. **Detects** resources in your state that have schema version mismatches
4. **Re-imports** those resources from the cloud provider (AWS, Azure, GCP, etc.)
5. **Updates** the state file with the correct schema versions
6. **Preserves** timeout configurations and metadata

## Installation

### From Source

```bash
git clone https://github.com/ishaankalra/terraform-state-downgrader.git
cd terraform-state-downgrader
go build -o terraform-state-downgrader .
```

### Using Go Install

```bash
go install github.com/ishaankalra/terraform-state-downgrader@latest
```

## Usage

### Prerequisites

1. Terraform must be installed and in your PATH
2. You must have run `terraform init` in your project directory
3. Your cloud provider credentials must be configured (AWS CLI, Azure CLI, etc.)

### Basic Usage

```bash
# Show what would change (dry run)
terraform-state-downgrader plan

# Apply the changes
terraform-state-downgrader apply
```

### With Custom Paths

```bash
# Specify custom config directory and state file
terraform-state-downgrader plan \
  --config-dir /path/to/terraform \
  --state-file /path/to/terraform.tfstate

# Apply with custom backup location
terraform-state-downgrader apply \
  --backup /path/to/backup.tfstate
```

## Example Workflow

### Scenario: Accidentally Upgraded AWS Provider

```hcl
# You had this:
terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 3.74"
    }
  }
}

# But accidentally upgraded to v6 and applied:
# version = "~> 6.0"
```

### Step 1: Downgrade Provider Version

```hcl
# Change back to v3 in your terraform config
terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 3.74"
    }
  }
}
```

### Step 2: Re-initialize

```bash
terraform init -upgrade
```

### Step 3: Run This Tool

```bash
# See what would change
terraform-state-downgrader plan

# Output:
# Analyzing configuration...
#   ✓ Parsed .terraform.lock.hcl (1 providers)
#   ✓ Reading: terraform.tfstate
#   ✓ Running: terraform providers
#   ✓ Running: terraform providers schema -json
#
# Providers from lock file:
#   • registry.terraform.io/hashicorp/aws v3.74.0
#
# Resources analyzed: 47
#
# Schema version mismatches: 8 resources
#
# registry.terraform.io/hashicorp/aws:
#   Version: 3.74.0
#
#   • aws_instance.web[0]
#     State schema: v1 → Target schema: v0
#     Resource ID: i-1234567890abcdef0
#     ⚠️  DOWNGRADE REQUIRED (v1 → v0)
#     Action: Re-import from cloud provider
#   ...

# Apply the changes
terraform-state-downgrader apply

# Output:
# Creating backup: terraform.tfstate.backup-1709400000
# Loading providers...
# Re-importing resources (8 total):
#   [1/8] ✓ aws_instance.web[0] (0.8s)
#   [2/8] ✓ aws_db_instance.main (1.2s)
#   ...
# ✓ Success! 8 resources downgraded
```

### Step 4: Verify

```bash
terraform plan
# Should show no changes if everything worked correctly
```

## Data Sources

The tool combines information from multiple sources:

1. **`.terraform.lock.hcl`** - Provider versions currently installed
2. **`terraform providers`** - Providers required by configuration (parses tree output)
3. **`terraform.tfstate`** - Current schema versions and resource-to-provider mappings
4. **`terraform providers schema -json`** - Target schema versions for each resource type

## Features

- ✅ **Provider-agnostic**: Works with any Terraform provider (AWS, Azure, GCP, Kubernetes, etc.)
- ✅ **Safe**: Creates automatic backups before making changes
- ✅ **Smart**: Preserves timeout configurations
- ✅ **Fast**: Only processes resources that need downgrading
- ✅ **Informative**: Clear output showing what will change

## Limitations

- **Local state only**: Does not support remote backends (S3, Terraform Cloud, etc.)
  - Workaround: Pull state locally, run tool, push back
- **Requires valid configuration**: Your Terraform config must be valid
- **Resources must exist**: Resources must still exist in the cloud provider
- **Read permissions required**: You need permissions to read resources from cloud

## How It Compares to Manual Approaches

### Manual State Surgery (Don't Do This!)

```bash
# ❌ Manual editing is error-prone and dangerous
vim terraform.tfstate
# Change schema_version: 6 → 3 (but what about new fields?)
```

### Using terraform import (Tedious)

```bash
# ❌ Manual import for each resource
terraform state rm aws_instance.web
terraform import aws_instance.web i-1234567890
# Repeat for 100+ resources... 😱
```

### Using This Tool (Easy!)

```bash
# ✅ Automatic, safe, and fast
terraform-state-downgrader apply
```

## Troubleshooting

### "Lock file not found"

```bash
Error: lock file not found at .terraform.lock.hcl. Run 'terraform init' first
```

**Solution**: Run `terraform init` in your Terraform directory.

### "Resource not found during re-import"

```bash
Error: Resource aws_instance.deleted with ID i-xyz789 was not found
```

**Solution**: The resource was deleted outside Terraform. Remove it from your configuration or restore it in the cloud provider.

### "Permission denied"

**Solution**: Ensure your cloud provider credentials have read access to all resources.

## Development

### Building from Source

```bash
git clone https://github.com/ishaankalra/terraform-state-downgrader.git
cd terraform-state-downgrader
go mod download
go build -o terraform-state-downgrader .
```

### Running Tests

```bash
go test ./...
```

### Project Structure

```
terraform-state-downgrader/
├── main.go                      # CLI entry point
├── cmd/
│   ├── root.go                 # Root command
│   ├── plan.go                 # Plan command
│   └── apply.go                # Apply command
├── internal/
│   ├── analysis/               # Mismatch detection
│   ├── config/                 # Configuration parsers
│   ├── state/                  # State file operations
│   ├── provider/               # Provider loading & re-import
│   └── output/                 # Output formatting
└── README.md
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

MIT License - see LICENSE file for details.

## Author

Ishaan Kalra - [@ishaankalra](https://github.com/ishaankalra)

## Acknowledgments

- HashiCorp Terraform team for the excellent tooling and plugin ecosystem
- The open-source community for inspiration and support