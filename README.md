# terraform-provider-podplane

Podplane OpenTofu/Terraform provider:

- creates a Netsy bootstrap snapshot file from a Podplane seed file by using Podplane's `netsyseed` package, which uploads the generated file to S3 or GCS as `bootstrap.netsy` using native cloud SDKs and create-only preconditions, so existing Netsy state and existing bootstrap files are never overwritten.

- renders Podplane VM userdata from pinned vmconfig manifest JSON using the same canonical Go template used for local clusters in the Podplane CLI.

- creates or adopts the dedicated workload CA key through a state-safe resource that never places private key bytes in configuration or state.

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

### Workload CA Key

`podplane_workload_ca_key` creates or adopts exactly one unencrypted PKCS#8 Ed25519 key at the canonical name derived from `provider` and `key_prefix`:

- `aws_secrets_manager`: `/<key_prefix>/workload-ca-key`
- `aws_ssm`: `/<key_prefix>/workload-ca-key` as a `SecureString`
- `gcp_secret_manager`: `<key_prefix>_workload-ca-key` in `project`

For AWS, `region` and `profile` can select the SDK configuration to use. For Google Cloud, `project` identifies the project in which the secret is stored.

The resource is designed to keep the private key out of Terraform state. It generates the key inside the provider and writes it directly to the selected secret backend. Terraform records only enough information to recognize the key later: its backend location, backend version, and public-key fingerprint.

The workload CA key is a long-lived cluster identity, so the provider will not silently replace or rotate it. During creation it uses the backend's create-only operation. If another provisioning process created the same key first, the provider adopts that key after checking that it is a valid Ed25519 private key. On later refreshes, it reads only secret metadata and reports an error if the key has disappeared or its current version has changed. Recover the original version instead of recreating the resource when this happens.

Changing any resource argument requires replacement, but automatic key replacement is intentionally unsupported. Import is also unsupported. Removing the resource from configuration removes it from Terraform state without deleting the key from the secret backend; retiring or rotating a workload CA must be an explicit operational procedure.

Google Secret Manager creates a secret and its first version in two separate operations. If provisioning is interrupted between them, an empty secret can remain. The provider will not add a key to that existing empty secret because another provisioning process could still be using it. Confirm that no other provisioning process is active, delete the empty secret manually, and apply again.

### Netsy seed

`podplane_netsy_seed_s3`:

- `cluster_config_path`
- `seed_path`
- `values_content` (optional inline YAML or JSON)
- `values_file` (optional user-authored YAML or JSON file)
- `bucket`
- `prefix` (optional)
- `region`
- `profile`

`podplane_netsy_seed_gcs`:

- `cluster_config_path`
- `seed_path`
- `values_content` (optional inline YAML or JSON)
- `values_file` (optional user-authored YAML or JSON file)
- `bucket`
- `prefix` (optional)
- `project`

The provider:

- generates snapshots in-process using Podplane's `netsyseed` package
- resolves `cluster.seed.name`/`version` from the published seeds manifest and verifies the file against `cluster.seed.digest`; `seed_path` may point to a custom Podplane seed file
- merges values with precedence: provider-derived defaults < `values_content` < `values_file`
- fails if the target prefix already contains Netsy state
- uploads with S3 `If-None-Match: *` or GCS `DoesNotExist` preconditions so existing Netsy state is never overwritten

Seed resources create the initial state for a new cluster. After Netsy has been initialized, changing a seed setting will not overwrite its existing data. To reconfigure components in a running cluster, update their live values instead.

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
