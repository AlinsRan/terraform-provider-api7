HOSTNAME     = registry.terraform.io
NAMESPACE    = api7
NAME         = api7
BINARY       = terraform-provider-$(NAME)
VERSION      = 0.1.0
OS_ARCH      = $(shell go env GOOS)_$(shell go env GOARCH)

INSTALL_DIR  = ~/.terraform.d/plugins/$(HOSTNAME)/$(NAMESPACE)/$(NAME)/$(VERSION)/$(OS_ARCH)

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
	oapi-codegen --config oapi-codegen.yaml ../openapi-subset.yaml

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
