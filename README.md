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

## Reporting bugs and requesting features

Think you found a bug? Please report all Crucible bugs — including bugs for the individual Crucible apps — in the [cmu-sei/crucible issue tracker](https://github.com/cmu-sei/crucible/issues).

Include as much detail as possible including steps to reproduce, specific app involved, and any error messages you may have received.

Have a good idea for a new feature? Submit all new feature requests through the [cmu-sei/crucible issue tracker](https://github.com/cmu-sei/crucible/issues).

Include the reasons why you're requesting the new feature and how it might benefit other Crucible users.

## License

Copyright 2022 Carnegie Mellon University. See the [LICENSE.md](./LICENSE.md) file for details.
