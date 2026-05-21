# 维护指南

## 项目结构

```
terraform-provider-api7/
├── main.go                          # Provider 入口
├── oapi-codegen.yaml                # 客户端生成配置
├── GNUmakefile                      # 构建目标
├── scripts/
│   └── extract-openapi.py          # 从完整规范自动提取子集
├── openapi-subset.yaml              # 自动生成，勿手动修改
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

当 API7 EE 的 API 发生变更时，只需两步：

**Step 1：在 api7ee-3-control-plane 仓库执行 `make openapi` 生成最新规范**

```bash
# 在 api7ee-3-control-plane 仓库中
make openapi
```

**Step 2：在本仓库重新生成客户端**

```bash
# 默认从 ../api7ee-3-control-plane 读取规范
make generate

# 如果仓库路径不同，通过环境变量指定
API7_SPEC=/path/to/openapi.generated.yaml make generate
```

`make generate` 会自动完成：
1. 从完整规范中提取 Provider 所需的 8 条路径（`scripts/extract-openapi.py`）
2. 用 oapi-codegen 重新生成 `internal/client/client.gen.go`

**Step 3：修复编译错误（如有）**

```bash
go build ./...
```

API 变更引起的常见编译错误：

| 错误 | 原因 | 处理方式 |
|------|------|---------|
| `undefined: client.XxxJSONRequestBody` | 生成的类型名变了 | 搜索新类型名替换 |
| `cannot use X as type Y` | 字段类型变更 | 调整 resource 文件中的类型转换 |
| `apiResp.JSON200.Value.XxxField undefined` | 响应结构变更 | 更新 `build*ModelFromResponse` 函数 |

**Step 4：验证**

```bash
make install
cd examples/quickstart && terraform plan
```

---

## 新增资源支持的 Operation

`scripts/extract-openapi.py` 中的 `OPERATION_IDS` 集合控制哪些 operation 会被提取进 subset。
新增资源时，在该集合里加入对应的 operation ID，再重新 `make generate` 即可。

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

> 注意：`Computed: true` 仅适用于服务端会自动填充的字段。纯用户管理的字段（如 `labels`）不要加 `Computed`，否则删除时 Terraform 不会产生 diff。

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
# 1. 在 scripts/extract-openapi.py 的 OPERATION_IDS 中加入 SSL 相关 operation ID
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
