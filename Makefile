HOSTNAME     = registry.terraform.io
NAMESPACE    = api7
NAME         = api7
BINARY       = terraform-provider-$(NAME)
VERSION      = 0.1.0
OS_ARCH      = $(shell go env GOOS)_$(shell go env GOARCH)

INSTALL_DIR  = ~/.terraform.d/plugins/$(HOSTNAME)/$(NAMESPACE)/$(NAME)/$(VERSION)/$(OS_ARCH)

# 完整 OpenAPI 规范路径，根据实际位置设置：
#   API7_SPEC=/path/to/api7ee-3-control-plane/internal/pkg/consts/manifests/openapi.generated.yaml make generate
API7_SPEC    ?= /workspace/api7/api7ee-3-control-plane/internal/pkg/consts/manifests/openapi.generated.yaml

default: build

# --------------------------------------------------------------------------
# 代码生成（完整流程）
#
#   make generate API7_SPEC=/path/to/openapi.generated.yaml
#
# 分步说明：
#   1. update-openapi      从完整 spec 提取 oapi-codegen 需要的子集
#   2. gen-client          oapi-codegen 生成类型安全的 HTTP 客户端
#   3. flatten-openapi     展平 allOf/oneOf，tfplugingen-openapi 不支持多层嵌套
#   4. gen-code-spec       tfplugingen-openapi 生成中间格式 provider_code_spec.json
#   5. gen-schema          tfplugingen-framework 生成含 validator 的 Schema + Types
# --------------------------------------------------------------------------

.PHONY: generate
generate: update-openapi gen-client flatten-openapi gen-code-spec gen-schema

.PHONY: update-openapi
update-openapi:
	python3 scripts/extract-openapi.py --spec $(API7_SPEC) --out openapi-subset.yaml

.PHONY: gen-client
gen-client:
	oapi-codegen --config oapi-codegen.yaml openapi-subset.yaml

.PHONY: flatten-openapi
flatten-openapi:
	python3 scripts/flatten-openapi.py $(API7_SPEC) openapi-flat.yaml

.PHONY: gen-code-spec
gen-code-spec:
	tfplugingen-openapi generate \
		--config generator_config.yml \
		--output provider_code_spec.json \
		openapi-flat.yaml

.PHONY: gen-schema
gen-schema:
	tfplugingen-framework generate resources \
		--input provider_code_spec.json \
		--output internal/provider/generated
	# tfplugingen-framework bug: consumer 的 PluginsValue 因 request+response 各生成一份导致重复声明，截断去重
	head -n 1049 internal/provider/generated/resource_consumer/consumer_resource_gen.go \
		> internal/provider/generated/resource_consumer/consumer_resource_gen.go.tmp
	mv internal/provider/generated/resource_consumer/consumer_resource_gen.go.tmp \
		internal/provider/generated/resource_consumer/consumer_resource_gen.go

# --------------------------------------------------------------------------

.PHONY: build
build:
	go build -o $(BINARY)

.PHONY: install
install: build
	mkdir -p $(INSTALL_DIR)
	cp $(BINARY) $(INSTALL_DIR)/$(BINARY)_v$(VERSION)

.PHONY: fmt
fmt:
	gofmt -w .
	terraform fmt -recursive examples/

.PHONY: test
test:
	go test ./... -v

.PHONY: clean
clean:
	rm -f $(BINARY)
	rm -f openapi-subset.yaml openapi-flat.yaml provider_code_spec.json
	rm -rf internal/provider/generated
