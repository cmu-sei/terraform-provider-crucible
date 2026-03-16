---
page_title: "crucible_player_view Resource"
description: |-
  Manages a view and its teams and applications in the Crucible Player API.
---

# crucible_player_view

Manages views in Crucible's Player API, including the teams and applications within them.

## Example Usage

```hcl
resource "crucible_player_view" "example" {
  name              = "example"
  description       = "This was created from Terraform!"
  status            = "Active"
  create_admin_team = true

  application {
    name               = "testApp"
    embeddable         = "false"
    load_in_background = "true"
  }

  team {
    name = "test_team"
    role = "SomeRole"

    user {
      user_id = "6fb5b293-668b-4eb6-b614-dfdd6b0e0acf"
    }

    app_instance {
      name          = "testApp"
      display_order = 0
    }
  }
}
```

## Argument Reference

### View

- `name` - (Required) The name of this view.
- `description` - (Optional) A description for this view.
- `status` - (Optional) The status of this view. Defaults to `"Active"`.
- `create_admin_team` - (Optional) Whether to automatically create an Admin team. Defaults to `true`.

### Applications

The `application` block is optional and repeatable. Applications should be placed in alphabetical order by name to avoid unnecessary state changes.

~> Due to a quirk in Terraform's type system, values for `embeddable` and `load_in_background` must be wrapped in quotes (e.g., `"false"`).

- `name` - (Required) The name of this application.
- `app_id` - (Computed) The UUID of this application, generated internally.
- `v_id` - (Computed) The UUID of the view this application belongs to.
- `url` - (Optional) A URL to associate with this application.
- `icon` - (Optional) A string pointing to the icon for this application.
- `embeddable` - (Optional) Whether this application is embeddable.
- `load_in_background` - (Optional) Whether this application should load in the background.
- `app_template_id` - (Optional) The UUID of an application template to inherit from.

### Teams

The `team` block is optional and repeatable. Teams should be placed in alphabetical order by name to avoid unnecessary state changes. Users within teams should also be ordered alphabetically by `user_id`, and app instances by `name`.

- `name` - (Required) The name of this team.
- `team_id` - (Computed) The UUID of this team, assigned by the API.
- `role` - (Optional) The name of the role this team falls under. Defaults to `"View Member"`.
- `permissions` - (Optional) A list of permission IDs for this team.

#### `user` block (nested inside `team`)

- `user_id` - (Required) The UUID of the user.
- `role` - (Optional) The name of a role to assign to this user within the team.

#### `app_instance` block (nested inside `team`)

- `name` - (Required) The name of the application to instantiate. Must match an `application` block's name.
- `display_order` - (Optional) The display order of this application within the team. Defaults to `0`.
- `id` - (Computed) The UUID of this application instance.

## Attribute Reference

- `id` - The UUID of the view.
