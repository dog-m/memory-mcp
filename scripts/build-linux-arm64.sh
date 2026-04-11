#!/bin/bash
set -e
cd ..

GOOS=linux
GOARCH=arm64
CGO_ENABLED=0
BUILD_DIR=build
OUTPUT_NAME="${BUILD_DIR}/memory-mcp"

mkdir -p "${BUILD_DIR}"

echo "Building memory-mcp for Linux arm64..."
go build -o "${OUTPUT_NAME}" ./src

echo "Build successful: ${OUTPUT_NAME}"
