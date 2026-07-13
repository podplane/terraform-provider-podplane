# terraform-provider-podplane

Podplane OpenTofu/Terraform provider:

- creates a Netsy bootstrap snapshot file from a Podplane seed file by using Podplane's `netsyseed` package, which uploads the generated file to S3 or GCS as `bootstrap.netsy` using native cloud SDKs and create-only preconditions, so existing Netsy state and existing bootstrap files are never overwritten.

- renders Podplane VM userdata from pinned vmconfig manifest JSON using the same canonical Go template used for local clusters in the Podplane CLI.

## Data sources

`podplane_userdata` renders auditable userdata during planning without network or filesystem access inside the provider:

```hcl
data "podplane_userdata" "knc_arm64" {
  manifest_json                 = file("${path.module}/podplane.cluster.vmconfig.knc.arm64.json")
  deps_mirror_url               = "https://deps.podplane.dev"
  provider_kind                 = "aws"
  aws_account_id                = data.aws_caller_identity.current.account_id
  immutable_ssh_authorized_keys = var.immutable_ssh_authorized_keys
  enable_ssm                    = var.enable_ssm
}
```

The manifest and rendered content are intentionally retained in Terraform state. Mutable runtime configuration, including `SSH_AUTHORIZED_KEYS`, is not rendered into userdata.

## Resource contract

`podplane_netsy_seed_s3`:

- `cluster_config_path`
- `seed_path`
- `values_path`
- `bucket`
- `prefix` (optional)
- `region`
- `profile`

`podplane_netsy_seed_gcs`:

- `cluster_config_path`
- `seed_path`
- `values_path`
- `bucket`
- `prefix` (optional)
- `project`

The provider:

- generates snapshots in-process using Podplane's `netsyseed` package
- resolves `cluster.seed.name`/`version` from the published seeds manifest and verifies the file against `cluster.seed.digest`; `seed_path` may point to a custom Podplane seed file
- merges `values_path` when configured
- fails if the target prefix already contains Netsy state
- uploads with S3 `If-None-Match: *` or GCS `DoesNotExist` preconditions so existing Netsy state is never overwritten

Cloud credentials are resolved by the native SDKs. AWS uses the standard AWS SDK chain, optionally scoped by `region` and `profile`. GCS uses Application Default Credentials, optionally scoped by `project` for quota/billing.

## Releases

Terraform/OpenTofu installs providers as released binaries. This repository is tagged with standard SemVer tags such as `v0.1.0`; release builds inject the provider version with `-ldflags "-X main.providerVersion=<version>"` and publish Terraform Registry-compatible zip archives.

For Terraform Registry publishing, the GitHub repository must be public and named `terraform-provider-podplane` so the Registry can detect provider `podplane/podplane`.

## Local Development

Build the local provider binary:

```sh
make build
```

This writes the provider into `bin/`. To make OpenTofu or Terraform use that local binary instead of a published registry version, configure provider development overrides.

Edit OpenTofu `~/.tofurc` or Terraform `~/.terraformrc` with:

```hcl
provider_installation {
  dev_overrides {
    "podplane/podplane" = "$HOME/Workspace/podplane/terraform-provider-podplane/bin"
  }

  direct {}
}
```

The override path is the provider binary directory, not the binary itself. Terraform/OpenTofu will warn that development overrides are active; this is expected.

After configuring the override, run the Podplane CLI or generated Terraform/OpenTofu files normally. Remove the override when testing published provider installs.
