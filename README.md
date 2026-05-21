# terraform-provider-api7

Terraform Provider for [API7 Enterprise](https://api7.ai), built with [oapi-codegen](https://github.com/oapi-codegen/oapi-codegen) + [terraform-plugin-framework](https://github.com/hashicorp/terraform-plugin-framework).

## Supported Resources

| Resource | Create | Read | Update | Delete | Import |
|----------|--------|------|--------|--------|--------|
| `api7_consumer` | ✅ | ✅ | ✅ | ✅ | ✅ |
| `api7_service`  | ✅ | ✅ | ✅ | ✅ | ✅ |
| `api7_route`    | ✅ | ✅ | ✅ | ✅ | ✅ |

## Requirements

- Terraform >= 1.0
- Go >= 1.21 (build only)
- API7 Enterprise 3.x

## Quick Start

```bash
git clone https://github.com/AlinsRan/terraform-provider-api7.git
cd terraform-provider-api7
make install
```

Then configure `~/.terraformrc` to use the local mirror:

```hcl
provider_installation {
  filesystem_mirror {
    path    = "/home/<YOUR_USER>/.terraform.d/plugins"
    include = ["registry.terraform.io/api7/api7"]
  }
  direct {
    exclude = ["registry.terraform.io/api7/api7"]
  }
}
```

See [`examples/provider/provider.tf`](examples/provider/provider.tf) for a minimal provider configuration.

## Documentation

- [Provider configuration](docs/index.md)
- [api7_consumer](docs/resources/consumer.md)
- [api7_service](docs/resources/service.md)
- [api7_route](docs/resources/route.md)
- [Maintenance guide](docs/guides/maintenance.md)
