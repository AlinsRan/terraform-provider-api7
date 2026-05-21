HOSTNAME     = registry.terraform.io
NAMESPACE    = api7
NAME         = api7
BINARY       = terraform-provider-$(NAME)
VERSION      = 0.1.0
OS_ARCH      = $(shell go env GOOS)_$(shell go env GOARCH)

INSTALL_DIR  = ~/.terraform.d/plugins/$(HOSTNAME)/$(NAMESPACE)/$(NAME)/$(VERSION)/$(OS_ARCH)

# 完整 OpenAPI 规范路径（可通过环境变量覆盖）
API7_SPEC    ?= ../api7ee-3-control-plane/internal/pkg/consts/manifests/openapi.generated.yaml

default: build

.PHONY: build
build:
	go build -o $(BINARY)

.PHONY: install
install: build
	mkdir -p $(INSTALL_DIR)
	cp $(BINARY) $(INSTALL_DIR)/

.PHONY: generate
generate:
	python3 scripts/extract-openapi.py --spec $(API7_SPEC) --out openapi-subset.yaml
	oapi-codegen --config oapi-codegen.yaml openapi-subset.yaml

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
