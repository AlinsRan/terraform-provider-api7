# api7_route

Manages a route under a published service in API7 EE.

## Example Usage

```terraform
resource "api7_route" "example" {
  name       = "get-anything"
  service_id = api7_service.example.id
  paths      = ["/anything/*"]
  methods    = ["GET"]
}
```

## Schema

### Required

- `name` (String) — Route name.
- `service_id` (String) — ID of the parent `api7_service`.
- `paths` (List of String) — URI paths (supports wildcards).

### Optional

- `methods` (List of String) — HTTP methods (e.g. `GET`, `POST`).
- `plugins` (String, JSON) — Plugin configuration as a JSON string.

### Read-Only

- `id` (String) — Route identifier.
