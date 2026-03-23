---
page_title: "Provider: Crucible"
description: |-
  The Crucible provider enables Terraform to manage resources within the Crucible cybersecurity training and simulation platform.
---

# Crucible Provider

The Crucible provider enables [Terraform](https://www.terraform.io/) to manage resources within the [Crucible](https://cmu-sei.github.io/crucible/) cybersecurity training and simulation platform, developed by Carnegie Mellon University's Software Engineering Institute (SEI).

## Resources

- [`crucible_player_virtual_machine`](resources/player_virtual_machine.md) — Manage virtual machines in the VM API
- [`crucible_player_view`](resources/player_view.md) — Manage views, teams, and applications in the Player API
- [`crucible_player_application_template`](resources/player_application_template.md) — Manage application templates in the Player API
- [`crucible_player_user`](resources/player_user.md) — Manage users in the Player API
- [`crucible_player_view_network`](resources/player_view_network.md) — Manage allowed team networks in the VM API
- [`crucible_vlan`](resources/vlan.md) — Acquire and release VLANs in the Caster API

## Authentication

The provider authenticates using OAuth2 resource owner password credentials. Credentials can be supplied via environment variables or directly in the provider block.

### Environment Variables

```shell
export SEI_CRUCIBLE_USERNAME="<your username>"
export SEI_CRUCIBLE_PASSWORD="<your password>"
export SEI_CRUCIBLE_AUTH_URL="<the url to the authentication service>"
export SEI_CRUCIBLE_TOK_URL="<the url to the token endpoint>"
export SEI_CRUCIBLE_CLIENT_ID="<your client ID>"
export SEI_CRUCIBLE_CLIENT_SECRET="<your client secret>"
export SEI_CRUCIBLE_CLIENT_SCOPES='["scope1","scope2"]'
export SEI_CRUCIBLE_VM_API_URL="<the url to the VM API>"
export SEI_CRUCIBLE_PLAYER_API_URL="<the url to the Player API>"
export SEI_CRUCIBLE_CASTER_API_URL="<the url to the Caster API>"
```

### Provider Block

```hcl
provider "crucible" {
  username       = "<your username>"
  password       = "<your password>"
  auth_url       = "<the url to the authentication service>"
  token_url      = "<the url to the token endpoint>"
  client_id      = "<your client ID>"
  client_secret  = "<your client secret>"
  client_scopes  = ["scope1", "scope2"]
  vm_api_url     = "<the url to the VM API>"
  player_api_url = "<the url to the Player API>"
  caster_api_url = "<the url to the Caster API>"
}
```

## Argument Reference

- `username` - (Required) Username for authentication. Can be set via `SEI_CRUCIBLE_USERNAME`.
- `password` - (Required) Password for authentication. Can be set via `SEI_CRUCIBLE_PASSWORD`.
- `auth_url` - (Required) URL to the authentication service. Can be set via `SEI_CRUCIBLE_AUTH_URL`.
- `token_url` - (Required) URL to the token endpoint. Can be set via `SEI_CRUCIBLE_TOK_URL`.
- `client_id` - (Required) OAuth2 client ID. Can be set via `SEI_CRUCIBLE_CLIENT_ID`.
- `client_secret` - (Required) OAuth2 client secret. Can be set via `SEI_CRUCIBLE_CLIENT_SECRET`.
- `client_scopes` - (Optional) List of OAuth2 scopes. Can be set via `SEI_CRUCIBLE_CLIENT_SCOPES`.
- `vm_api_url` - (Required) URL to the VM API. Can be set via `SEI_CRUCIBLE_VM_API_URL`.
- `player_api_url` - (Required) URL to the Player API. Can be set via `SEI_CRUCIBLE_PLAYER_API_URL`.
- `caster_api_url` - (Required) URL to the Caster API. Can be set via `SEI_CRUCIBLE_CASTER_API_URL`.
