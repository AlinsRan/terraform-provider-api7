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
- `upstream` (Attributes) — Upstream configuration.
  - `nodes` (List of Attributes, Required) — Backend nodes.
    - `host` (String, Required)
    - `port` (Number, Required)
    - `weight` (Number, Required)

### Optional

- `desc` (String) — Description.
- `upstream.scheme` (String) — Protocol: `http`, `https`, `grpc`, `grpcs`.
- `upstream.type` (String) — Load balancing algorithm: `roundrobin`, `chash`, `least_conn`, `ewma`.
- `plugins` (String, JSON) — Plugin configuration as a JSON string.

### Read-Only

- `id` (String) — Service identifier.

## Import

```bash
terraform import api7_service.<name> <service-id>
```

## Known Limitations

The API7 EE Service GET response does not include upstream fields. After `terraform import`, run `terraform apply` once to sync the upstream configuration from state to the gateway.
