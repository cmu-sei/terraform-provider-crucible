---
page_title: "crucible_player_application Resource"
description: |-
  Manages a standalone application in a Crucible Player view.
---

# crucible_player_application

Manages one application in a view configured for standalone child management.

> **Warning:** Do not use this resource with nested `application` blocks on the same view. Mixed ownership is unsupported and can cause perpetual differences, overwritten settings, or deletion.

```hcl
resource "crucible_player_application" "terminal" {
  view_id            = crucible_player_view.example.id
  name               = "terminal"
  url                = "https://terminal.example.test"
  embeddable         = false
  load_in_background = false
}
```

## Arguments

- `view_id` - (Required, Forces replacement) Owning view UUID.
- `name` - (Required) Application name.
- `url` - (Optional) HTTP or HTTPS application URL.
- `icon` - (Optional) Icon value.
- `embeddable` - (Optional) Whether the application can be embedded.
- `load_in_background` - (Optional) Whether it loads in the background.
- `application_template_id` - (Optional) Application-template UUID.

## Attributes

- `id` - Application UUID.

## Import

```shell
terraform import crucible_player_application.terminal <application_uuid>
```

Deleting the parent view also deletes this application. Reference the view
resource in `view_id` so Terraform destroys the application before its parent.

## `for_each`

```hcl
resource "crucible_player_application" "apps" {
  for_each = {
    terminal = "https://terminal.example.test"
    guide    = "https://guide.example.test"
  }

  view_id = crucible_player_view.example.id
  name    = each.key
  url     = each.value
}
```
