#!/bin/bash

# Set default values
IMAGE_NAME="go-loan-service"
IMAGE_TAG="latest"
DOCKER_REGISTRY=""

# Parse command line arguments
while [[ $# -gt 0 ]]; do
  case $1 in
    --registry=*)
      DOCKER_REGISTRY="${1#*=}"
      shift
      ;;
    --tag=*)
      IMAGE_TAG="${1#*=}"
      shift
      ;;
    *)
      echo "Unknown option: $1"
      exit 1
      ;;
  esac
done

# Build the Docker image
echo "Building Docker image..."
docker build -t ${DOCKER_REGISTRY}${IMAGE_NAME}:${IMAGE_TAG} .

# Check if build was successful
if [ $? -eq 0 ]; then
  echo "Docker image built successfully: ${DOCKER_REGISTRY}${IMAGE_NAME}:${IMAGE_TAG}"
else
  echo "Docker build failed"
  exit 1
fi 