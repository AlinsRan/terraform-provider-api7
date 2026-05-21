# api7_service

Manages a published service in API7 EE.

## Example Usage

```terraform
resource "api7_service" "example" {
  name = "httpbin"
  desc = "HTTPBin service"

  upstream = {
    nodes = [
      {
        host   = "httpbin.org"
        port   = 80
        weight = 100
      }
    ]
    scheme = "http"
    type   = "roundrobin"
  }
}
```

## Schema

### Required

- `name` (String) — Service name.

### Optional

- `desc` (String) — Description.
- `upstream` (Attributes) — Upstream configuration.
  - `nodes` (List of Attributes, Required) — Upstream nodes.
    - `host` (String, Required)
    - `port` (Number, Required)
    - `weight` (Number, Required)
  - `scheme` (String) — Protocol scheme (`http`, `https`, `grpc`, `grpcs`).
  - `type` (String) — Load balancing algorithm (`roundrobin`, `chash`, `ewma`, `least_conn`).
- `plugins` (String, JSON) — Plugin configuration as a JSON string.

### Read-Only

- `id` (String) — Service identifier.
