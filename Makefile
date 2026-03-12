# Makefile for the 32Hybrid AVD system
# Targets: proto, build, clean, test, iso, run-iso

GOPATH          ?= $(shell go env GOPATH)
PROTOC          ?= protoc
PROTO_GEN_GO    ?= $(GOPATH)/bin/protoc-gen-go
PROTO_GEN_GRPC  ?= $(GOPATH)/bin/protoc-gen-go-grpc
PROTO_DIR       := proto
GEN_DIR         := gen
CMD_DIRS        := ./cmd/controlplane ./cmd/runner ./cmd/avdclient
BIN_DIR         := bin
DIST_DIR        := dist
ISO_OUTPUT      := $(DIST_DIR)/hyper32.iso
ISO_SCRIPT      := iso/build.sh

.PHONY: all proto build clean test install-proto-tools iso run-iso check-iso-deps

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

## clean: remove generated binaries and ISO build artifacts
clean:
	rm -rf $(BIN_DIR) $(DIST_DIR) iso/build

# ─────────────────────────────────────────────
# ISO build
# ─────────────────────────────────────────────

## check-iso-deps: verify ISO build prerequisites are installed
check-iso-deps:
	@bash iso/scripts/check-deps.sh

## iso: build a bootable Hyper 32 live ISO -> dist/hyper32.iso
## The Go binaries are built first so they can be bundled into the image.
## Re-run 'make iso' after any Go changes to refresh the ISO.
iso: build
	@echo "Building Hyper 32 ISO..."
	@bash $(ISO_SCRIPT)
	@echo "ISO ready: $(ISO_OUTPUT)"

## run-iso: boot the ISO in QEMU (requires qemu-system-x86_64)
run-iso: $(ISO_OUTPUT)
	@command -v qemu-system-x86_64 >/dev/null 2>&1 || \
	    { echo "ERROR: qemu-system-x86_64 not found. Install qemu-system-x86."; exit 1; }
	qemu-system-x86_64 \
	    -m 512M \
	    -cdrom $(ISO_OUTPUT) \
	    -boot d \
	    -nographic \
	    -serial mon:stdio \
	    -no-reboot

$(ISO_OUTPUT):
	@$(MAKE) iso
