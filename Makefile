# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
BINARY_NAME=congo
ROOTFS_DIR=$(shell pwd)/rootfs

# Default target
all: build

# Build the application
build:
	@echo "Building $(BINARY_NAME)..."
	$(GOBUILD) -o $(BINARY_NAME) main.go

# Clean the binary
clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	rm -f $(BINARY_NAME)

# Ensure rootfs is downloaded
ensure_rootfs:
	@./scripts/download_rootfs.sh

# Run a command in the container
# Pass arguments to the congo command using ARGS
# Example: make run ARGS="--hostname my-alpine /bin/echo Hello from container"
run: build ensure_rootfs
	@if [ -z "$(ARGS)" ]; then \
		echo "Usage: make run ARGS=\"<congo arguments>\""; \
		echo "Example: make run ARGS=\"--hostname my-alpine /bin/echo Hello\""; \
		exit 1; \
	fi
	@echo "Running congo with rootfs: $(ROOTFS_DIR)"
	@sudo ./$(BINARY_NAME) run $(ARGS) --rootfs $(ROOTFS_DIR)

# Get an interactive shell in a container
# Example: make shell ARGS="<container-id>"
shell: build
	@if [ -z "$(ARGS)" ]; then \
		echo "Usage: make shell ARGS=\"<container-id>\""; \
		exit 1; \
	fi
	@sudo ./$(BINARY_NAME) shell $(ARGS)

# List all containers
ps: build
	@./$(BINARY_NAME) ps

.PHONY: all build clean run shell ps ensure_rootfs
