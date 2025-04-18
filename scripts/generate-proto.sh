#!/bin/bash

# Exit on error
set -e

# Generate Go code from proto files
protoc --go_out=. --go_opt=paths=source_relative \
    --go-grpc_out=. --go-grpc_opt=paths=source_relative \
    proto/sop/sop.proto

echo "Generated Go code from proto files" 