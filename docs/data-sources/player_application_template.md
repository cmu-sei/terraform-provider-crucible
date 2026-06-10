---
page_title: "crucible_player_application_template Data Source"
description: |-
  Looks up an application template in the Crucible Player API by name.
---

# crucible_player_application_template

Looks up an application template in Crucible's Player API by name and exposes its `id` (and other fields). Use this to reference an existing template by its friendly name instead of hardcoding its UUID, for example when wiring an `application` block inside a `crucible_player_view`.

The Player API does not provide a lookup-by-name endpoint, so this data source lists all templates and filters client-side. Template names are not guaranteed to be unique; if more than one template matches the given name, the lookup fails. It also fails if no template matches.

## Example Usage

```hcl
data "crucible_player_application_template" "vms" {
  name = "Virtual Machines"
}

resource "crucible_player_view" "player" {
  name   = "Example View"
  status = "Active"

  application {
    name            = "VMs"
    app_template_id = data.crucible_player_application_template.vms.id
  }
}
```

## Argument Reference

- `name` - (Required) The name of the application template to look up.
- `case_insensitive` - (Optional) Whether to match the name ignoring case. Defaults to `false` (exact, case-sensitive match).

## Attribute Reference

- `id` - The UUID of the matched application template.
- `url` - The URL the application template points to.
- `icon` - The URL to the template's icon image.
- `embeddable` - Whether the template is embeddable.
- `load_in_background` - Whether the template loads in the background.
