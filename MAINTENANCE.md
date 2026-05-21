# 维护指南

## 项目结构

```
terraform-provider-api7/
├── main.go                          # Provider 入口
├── oapi-codegen.yaml                # 客户端生成配置
├── GNUmakefile                      # 构建目标
├── internal/
│   ├── client/
│   │   └── client.gen.go           # oapi-codegen 自动生成，勿手动修改
│   └── provider/
│       ├── provider.go
│       ├── consumer_resource.go
│       ├── service_resource.go
│       └── route_resource.go
└── examples/
    ├── provider/provider.tf
    └── resources/api7_{consumer,service,route}/resource.tf
```

## 更新 OpenAPI 规范

当 API7 EE 的 OpenAPI 规范发生变更时，需同步更新 `openapi-subset.yaml` 并重新生成客户端。

**Step 1：查看上游变更**

```bash
# 在 api7ee-3-control-plane 仓库中
git diff openapi/routers/consumer.yaml
git diff openapi/schemas/service.yaml
git diff openapi/schemas/route.yaml
```

**Step 2：更新 `openapi-subset.yaml`**

该文件是从完整规范手动裁剪的子集，仅包含 Provider 用到的 8 条路径。按实际变更同步对应的 request/response schema。

> 完整规范位于 `internal/pkg/consts/manifests/openapi.generated.yaml`（执行 `make openapi` 后生成）。

**Step 3：重新生成客户端**

```bash
make generate
```

**Step 4：修复编译错误**

```bash
go build ./...
```

常见错误：

| 错误 | 原因 | 处理方式 |
|------|------|---------|
| `undefined: client.XxxJSONRequestBody` | 生成的类型名变了 | 搜索新类型名替换 |
| `cannot use X as type Y` | 字段类型变更 | 调整 resource 文件中的类型转换 |
| `apiResp.JSON200.Value.XxxField undefined` | 响应结构变更 | 更新 `build*ModelFromResponse` 函数 |

**Step 5：验证**

```bash
go build ./...
make install
cd examples/provider && terraform plan
```

---

## 新增字段

以 Service 新增 `timeout` 字段为例：

**1. Schema 中加字段**（`internal/provider/service_resource.go`）：

```go
"timeout": schema.Int64Attribute{
    Optional:    true,
    Computed:    true,
    Description: "Upstream 超时时间（秒）。",
},
```

**2. Model struct 中加字段**：

```go
type ServiceResourceModel struct {
    // ...现有字段...
    Timeout types.Int64 `tfsdk:"timeout"`
}
```

**3. `buildServiceRequestBody` 中加字段**：

```go
if !m.Timeout.IsNull() && !m.Timeout.IsUnknown() {
    body["timeout"] = m.Timeout.ValueInt64()
}
```

**4. `buildServiceModelFromResponse` 中加字段**：

```go
if v, ok := m["timeout"].(float64); ok {
    state.Timeout = types.Int64Value(int64(v))
}
```

---

## 新增资源

以新增 `api7_ssl` 为例：

```bash
# 1. 在 openapi-subset.yaml 中添加 SSL 相关路径
# 2. 重新生成客户端
make generate

# 3. 参考现有文件创建新 resource
cp internal/provider/consumer_resource.go internal/provider/ssl_resource.go
# 修改类型名、字段、API 调用方法

# 4. 在 provider.go 的 Resources() 中注册 NewSSLResource
```

---

## 版本对应

| API7 EE | Provider | 说明 |
|---------|----------|------|
| 3.9.x   | 0.1.x    | 当前版本 |
| 3.10.x+ | 0.2.x+   | 按需更新 |
