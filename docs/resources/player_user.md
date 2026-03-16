---
page_title: "crucible_player_user Resource"
description: |-
  Manages a user in the Crucible Player API.
---

# crucible_player_user

Creates users within Crucible's Player API. This is distinct from a `user` block inside of a `team` within a view — the `team` user block assumes a user with the given ID already exists, whereas this resource _creates_ the user.

This is intended to be used in conjunction with an identity provider to create accounts and add corresponding users to Player. Once created, users can be referenced within teams and views.

## Example Usage

```hcl
resource "crucible_player_user" "example" {
  user_id = identity_account.example.global_id
  name    = regex("(.*)(@.*)", identity_account.example.username)[0]
  role    = "ExampleRole"
}
```

## Argument Reference

- `user_id` - (Required) The UUID to create this user with. Typically points to an identity provider account's GUID.
- `name` - (Required) The display name for this user.
- `role` - (Optional) A role to assign to this user.

## Attribute Reference

- `id` - The UUID of the user (same as `user_id`).
