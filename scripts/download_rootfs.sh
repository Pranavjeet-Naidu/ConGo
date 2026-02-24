#!/bin/bash

set -e

ROOTFS_DIR="rootfs"
ALPINE_VERSION="3.18.4"
ALPINE_TARBALL="alpine-minirootfs-${ALPINE_VERSION}-x86_64.tar.gz"
ALPINE_URL="https://dl-cdn.alpinelinux.org/alpine/v3.18/releases/x86_64/${ALPINE_TARBALL}"

# Check if rootfs directory already exists
if [ -d "$ROOTFS_DIR" ]; then
    echo "Rootfs directory already exists at '${ROOTFS_DIR}'. Skipping download."
    exit 0
fi

echo "Rootfs not found. Downloading Alpine Mini Rootfs..."

# Create a temporary directory for the download
TMP_DIR=$(mktemp -d)
trap 'rm -rf -- "$TMP_DIR"' EXIT

# Download the tarball
if command -v wget &> /dev/null; then
    wget -P "$TMP_DIR" "$ALPINE_URL"
else
    curl -L -o "$TMP_DIR/$ALPINE_TARBALL" "$ALPINE_URL"
fi

# Create the rootfs directory and extract the tarball
mkdir -p "$ROOTFS_DIR"
echo "Extracting rootfs to '${ROOTFS_DIR}'..."
tar -xzf "$TMP_DIR/$ALPINE_TARBALL" -C "$ROOTFS_DIR"

echo "Rootfs downloaded and extracted successfully."
