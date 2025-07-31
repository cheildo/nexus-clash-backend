#!/bin/bash

# Exit immediately if a command exits with a non-zero status.
set -e

# Define the base directory for proto files.
PROTO_DIR=./api/proto

# Generate Go gRPC code.
protoc --proto_path=${PROTO_DIR} \
       --go_out=./api/proto --go_opt=paths=source_relative \
       --go-grpc_out=./api/proto --go-grpc_opt=paths=source_relative \
       $(find ${PROTO_DIR} -name '*.proto')

echo "✅ Protobuf code generated successfully."