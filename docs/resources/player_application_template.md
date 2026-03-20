---
page_title: "crucible_player_application_template Resource"
description: |-
  Manages an application template in the Crucible Player API.
---

# crucible_player_application_template

Manages application templates in Crucible's Player API. Application templates are a distinct resource that exist independently of views and can be referenced by applications within views.

## Example Usage

```hcl
resource "crucible_player_application_template" "example" {
  name               = "Example"
  url                = "http://example.com"
  icon               = "https://example.com/icon.png"
  embeddable         = false
  load_in_background = false
}
```

## Argument Reference

- `name` - (Required) The name of the application template.
- `url` - (Optional) The URL this application template should point to.
- `icon` - (Optional) The URL to an image to use as the template's icon.
- `embeddable` - (Optional) Whether this template is embeddable.
- `load_in_background` - (Optional) Whether this template should load in the background.

## Attribute Reference

- `id` - The UUID of the application template.
