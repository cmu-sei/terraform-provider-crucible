# Terraform Provider Crucible

The Crucible Terraform Provider enables infrastructure-as-code management of resources within the [Crucible](https://cmu-sei.github.io/crucible/) cybersecurity training and simulation platform, developed by Carnegie Mellon University's Software Engineering Institute (SEI). It supports managing virtual machines, views, teams, application templates, users, VLANs, and view networks.

Full resource documentation is available on the [Terraform Registry](https://registry.terraform.io/providers/cmu-sei/crucible/latest).

## Development

This project uses [Task](https://taskfile.dev/) for build automation. Run `task` to see available tasks and descriptions.

### Getting started

1. Clone the repository
2. Run `task install` to build the provider and configure Terraform dev overrides
3. Create a `.tf` file with a `crucible` provider block (see [docs](https://registry.terraform.io/providers/cmu-sei/crucible/latest))
4. Run `terraform plan` and `terraform apply` — Terraform will use your local build automatically

`task install` writes a dev override file to `~/.terraform.d/crucible-dev.tfrc` that points Terraform at the locally built binary. Run `task uninstall` to revert.

### Running tests

- `task test` — fast unit/compile check. Acceptance steps are skipped unless `TF_ACC` is set.
- `task testacc` — full acceptance suite against a live Crucible stack (sets `TF_ACC=1`).

The connection settings and `TF_TEST_USER_ID` default to the local [crucible-development](https://github.com/cmu-sei/crucible) Aspire stack, so with that stack running `task testacc` works out of the box. Override the `TF_*` variables only when targeting another environment. The VM tests provision their own Player view and teams (no pre-seeded team GUIDs required) and stamp each VM with `TF_TEST_VM_USER_ID`; the Player VM API stores `user_id` without validating it against Keycloak, so the default works against any stack and you only need to override it to point at a specific real user. The `crucible_vlan` test acquires from Caster's system-wide default partition by default; point it elsewhere with the optional `TF_TEST_PARTITION_ID`, `TF_TEST_VLAN_ID`, and `TF_TEST_VLAN_ID_2` overrides (the two VLAN ids must exist and be free in the target partition). Set `TF_TEST_PROJECT_ID` to a Caster project GUID whose project has an assigned VLAN partition to opt into an additional `crucible_vlan` test that exercises the by-project acquire path; it skips when unset. Set `TF_TEST_UPGRADE=1` to additionally run the old→new provider upgrade tests (these download the published provider, so they need Terraform Registry access).

## Reporting bugs and requesting features

Think you found a bug? Please report all Crucible bugs — including bugs for the individual Crucible apps — in the [cmu-sei/crucible issue tracker](https://github.com/cmu-sei/crucible/issues).

Include as much detail as possible including steps to reproduce, specific app involved, and any error messages you may have received.

Have a good idea for a new feature? Submit all new feature requests through the [cmu-sei/crucible issue tracker](https://github.com/cmu-sei/crucible/issues).

Include the reasons why you're requesting the new feature and how it might benefit other Crucible users.

## License

Copyright 2022 Carnegie Mellon University. See the [LICENSE.md](./LICENSE.md) file for details.
