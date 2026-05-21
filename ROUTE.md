# api7_route

管理 API7 EE 中某个已发布服务下的路由。

## 示例

```terraform
resource "api7_route" "example" {
  name       = "get-anything"
  service_id = api7_service.example.id
  paths      = ["/anything/*"]
  methods    = ["GET"]
}
```

完整示例见 [`examples/resources/api7_route/resource.tf`](examples/resources/api7_route/resource.tf)。

## 参数说明

### 必填

- `name`（String）— 路由名称。
- `service_id`（String）— 所属 `api7_service` 的 ID，变更会触发资源重建。
- `paths`（List of String）— 匹配路径，支持通配符，如 `["/api/*"]`。

### 可选

- `methods`（List of String）— HTTP 方法，如 `["GET", "POST"]`，留空表示匹配所有方法。
- `plugins`（String，JSON）— 插件配置，JSON 字符串格式。

### 只读

- `id`（String）— 路由 ID。

## 导入

```bash
terraform import api7_route.<名称> <route-id>
```
