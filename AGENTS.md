# Podplane OpenTofu/Terraform Provider — Publication Repository

The Podplane OpenTofu/Terraform Provider source and tests live in the main
`podplane/podplane` repository: the entrypoint is
`cmd/terraform-provider-podplane/main.go`, implementation is `internal/tfprovider`,
and canonical registry documentation is `cmd/terraform-provider-podplane/docs.md`,
published here as `docs/index.md`.

This repository contains the registry manifest, signing public key, README,
and release publications. The source release workflow manages `docs/`,
`terraform-registry-manifest.json`, and `release-metadata.json`; edit their
source-repository counterparts instead. Do not add a second Go module or mirror
the implementation here.

- `make help`: list commands.
- `make setup`: install the pre-commit and canonical DCO commit-message hooks.
- `make precommit`: check publication metadata and hook syntax without writes.

Build and test in `podplane/podplane` using `make build` and `make test`.
Publishing is performed by that repository's coordinated release workflow.
Never replace published tags or release assets. Ask before any mutating Git
operation, and preserve existing staged/unstaged work.
