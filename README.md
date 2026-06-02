# terraform-provider-podplane

Podplane OpenTofu/Terraform provider.

This provider creates a Netsy bootstrap snapshot file from a Podplane seed file by using Podplane's `netsyseed` package.

It uploads the generated file to S3 or GCS as `bootstrap.netsy` using native cloud SDKs and create-only preconditions, so existing Netsy state and existing bootstrap files are never overwritten.

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
- resolves `cluster.seed.name`/`version` from the published seeds manifest, unless `seed_path` points to a custom Podplane seed file
- merges `values_path` when configured
- fails if the target prefix already contains Netsy state
- uploads with S3 `If-None-Match: *` or GCS `DoesNotExist` preconditions so existing Netsy state is never overwritten

Cloud credentials are resolved by the native SDKs. AWS uses the standard AWS SDK chain, optionally scoped by `region` and `profile`. GCS uses Application Default Credentials, optionally scoped by `project` for quota/billing.

## Releases

Terraform/OpenTofu installs providers as released binaries. This repository is tagged with standard SemVer tags such as `v1.0.0`; release builds inject the provider version with `-ldflags "-X main.providerVersion=<version>"` and publish Terraform Registry-compatible zip archives.

For Terraform Registry publishing, the GitHub repository must be public and named `terraform-provider-podplane` so the Registry can detect provider `podplane/podplane`.

For local development, build the binary yourself and use Terraform CLI `dev_overrides` for `podplane/podplane`.
