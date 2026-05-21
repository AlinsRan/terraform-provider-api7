# terraform-provider-api7

基于 [oapi-codegen](https://github.com/oapi-codegen/oapi-codegen) + [terraform-plugin-framework](https://github.com/hashicorp/terraform-plugin-framework) 构建的 [API7 Enterprise](https://api7.ai) Terraform Provider。

## 支持的资源

| 资源 | Create | Read | Update | Delete | Import |
|------|--------|------|--------|--------|--------|
| `api7_consumer` | ✅ | ✅ | ✅ | ✅ | ✅ |
| `api7_service`  | ✅ | ✅ | ✅ | ✅ | ✅ |
| `api7_route`    | ✅ | ✅ | ✅ | ✅ | ✅ |

## 环境要求

- Terraform >= 1.0
- Go >= 1.21（仅构建时需要）
- API7 Enterprise 3.x

## 快速开始

```bash
git clone https://github.com/AlinsRan/terraform-provider-api7.git
cd terraform-provider-api7
make install
```

配置 `~/.terraformrc` 使用本地 mirror：

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

使用示例见 [`examples/provider/provider.tf`](examples/provider/provider.tf)。

## 文档

- [Provider 配置](PROVIDER.md)
- [维护指南](MAINTENANCE.md)

## 示例

- [快速开始](examples/quickstart/README.md) — 创建 Consumer、Service、Route 的完整示例
