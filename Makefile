# Makefile for the 32Hybrid AVD system
# Targets: proto, build, clean, test

GOPATH          ?= $(shell go env GOPATH)
PROTOC          ?= protoc
PROTO_GEN_GO    ?= $(GOPATH)/bin/protoc-gen-go
PROTO_GEN_GRPC  ?= $(GOPATH)/bin/protoc-gen-go-grpc
PROTO_DIR       := proto
GEN_DIR         := gen
CMD_DIRS        := ./cmd/controlplane ./cmd/runner ./cmd/avdclient
BIN_DIR         := bin

.PHONY: all proto build clean test install-proto-tools

all: proto build

# ─────────────────────────────────────────────
# Proto generation
# ─────────────────────────────────────────────

## install-proto-tools: install protoc-gen-go and protoc-gen-go-grpc
install-proto-tools:
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

## proto: regenerate Go code from .proto files using protoc
proto: install-proto-tools
	@echo "Generating proto..."
	$(PROTOC) \
		--proto_path=$(PROTO_DIR) \
		--go_out=$(GEN_DIR) \
		--go_opt=paths=source_relative \
		$(PROTO_DIR)/common/v1/common.proto

	$(PROTOC) \
		--proto_path=$(PROTO_DIR) \
		--go_out=$(GEN_DIR) \
		--go_opt=paths=source_relative \
		--go-grpc_out=$(GEN_DIR) \
		--go-grpc_opt=paths=source_relative \
		--go-grpc_opt=require_unimplemented_servers=false \
		$(PROTO_DIR)/controlplane/v1/controlplane.proto

	$(PROTOC) \
		--proto_path=$(PROTO_DIR) \
		--go_out=$(GEN_DIR) \
		--go_opt=paths=source_relative \
		--go-grpc_out=$(GEN_DIR) \
		--go-grpc_opt=paths=source_relative \
		--go-grpc_opt=require_unimplemented_servers=false \
		$(PROTO_DIR)/runner/v1/runner.proto
	@echo "Proto generation complete."

## build: compile all binaries into bin/
build:
	@mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/controlplane ./cmd/controlplane
	go build -o $(BIN_DIR)/runner       ./cmd/runner
	go build -o $(BIN_DIR)/avdclient    ./cmd/avdclient
	@echo "Binaries written to $(BIN_DIR)/"

## test: run all Go tests
test:
	go test ./...

## clean: remove generated binaries
clean:
	rm -rf $(BIN_DIR)
