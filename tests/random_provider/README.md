# Random Provider Test Fixture

This directory contains a Terraform configuration using the `random` provider for integration testing.

## Setup

1. Initialize Terraform:
   ```bash
   terraform init
   ```

2. Apply the configuration:
   ```bash
   terraform apply -auto-approve
   ```

3. To simulate a schema version mismatch, manually edit `terraform.tfstate` and change some `schema_version` values to higher numbers.

## Resources

- `random_id.server` - Generates a random ID
- `random_string.suffix` - Generates a random string
- `random_password.db_password` - Generates a random password
- `random_pet.server_name` - Generates a random pet name
- `random_integer.priority` - Generates a random integer

## Testing the Tool

After setup, you can test the terraform-state-downgrader tool:

```bash
# From the root of the repository
./terraform-state-downgrader plan --config-dir ./tests/random_provider

# Apply the downgrade
./terraform-state-downgrader apply --config-dir ./tests/random_provider
```
