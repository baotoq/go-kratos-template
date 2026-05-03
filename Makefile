GOHOSTOS:=$(shell go env GOHOSTOS)
GOPATH:=$(shell go env GOPATH)
VERSION=$(shell git describe --tags --always)

ifeq ($(GOHOSTOS), windows)
    #the `find.exe` is different from `find` in bash/shell.
    #to see https://docs.microsoft.com/en-us/windows-server/administration/windows-commands/find.
    #changed to use git-bash.exe to run find cli or other cli friendly, caused of every developer has a Git.
    #Git_Bash= $(subst cmd\,bin\bash.exe,$(dir $(shell where git)))
    Git_Bash=$(subst \,/,$(subst cmd\,bin\bash.exe,$(dir $(shell where git))))
    INTERNAL_PROTO_FILES=$(shell $(Git_Bash) -c "find app -name *.proto -not -path '*/api/*.proto'")
    API_PROTO_FILES=$(shell $(Git_Bash) -c "find api -name *.proto")
else
    INTERNAL_PROTO_FILES=$(shell find app -name *.proto -not -path "*/api/*.proto")
    API_PROTO_FILES=$(shell find api -name *.proto)
endif

.PHONY: init
# init env (versions pinned to keep generated code reproducible across machines).
# kratos cmd subpackages have no semver tags — pin to a pseudo-version.
# protoc-gen-openapi: v0.6.3 is the newest tagged release of gnostic.
init:
	go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.11
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.5.1
	go install github.com/go-kratos/kratos/cmd/kratos/v2@v2.0.0-20260404020628-f149714c1d54
	go install github.com/go-kratos/kratos/cmd/protoc-gen-go-http/v2@v2.0.0-20260404020628-f149714c1d54
	go install github.com/google/gnostic/cmd/protoc-gen-openapi@v0.7.1
	go install github.com/google/wire/cmd/wire@v0.7.0

.PHONY: config
# generate internal proto
config:
	protoc --proto_path=. \
	       --proto_path=./third_party \
	       --go_out=paths=source_relative:. \
	       $(INTERNAL_PROTO_FILES)

.PHONY: api
# generate api proto
api:
	protoc --proto_path=./api \
	       --proto_path=./third_party \
	       --go_out=paths=source_relative:./api \
	       --go-http_out=paths=source_relative:./api \
	       --go-grpc_out=paths=source_relative:./api \
	       --openapi_out=fq_schema_naming=true,default_response=false:app/coffee \
	       $(API_PROTO_FILES)

.PHONY: build
# build
build:
	mkdir -p bin/ && go build -ldflags "-X main.Name=coffee -X main.Version=$(VERSION)" -o ./bin/coffee ./app/coffee/cmd/server

.PHONY: test
# test (requires Docker — testcontainers-go integration tests run alongside units)
test:
	go test -race -count=1 -v ./...

.PHONY: generate
# generate
generate:
	go generate ./...
	go mod tidy

.PHONY: dev
# start dev environment with tilt
dev:
	tilt up -- --continue

.PHONY: debug
# start dev environment, wait for debugger to attach
debug:
	tilt up

.PHONY: all
# generate all
all:
	$(MAKE) api
	$(MAKE) config
	$(MAKE) generate

# show help
help:
	@echo ''
	@echo 'Usage:'
	@echo ' make [target]'
	@echo ''
	@echo 'Targets:'
	@awk '/^[a-zA-Z\-\_0-9]+:/ { \
	helpMessage = match(lastLine, /^# (.*)/); \
		if (helpMessage) { \
			helpCommand = substr($$1, 0, index($$1, ":")-1); \
			helpMessage = substr(lastLine, RSTART + 2, RLENGTH); \
			printf "\033[36m%-22s\033[0m %s\n", helpCommand,helpMessage; \
		} \
	} \
	{ lastLine = $$0 }' $(MAKEFILE_LIST)

.DEFAULT_GOAL := help
