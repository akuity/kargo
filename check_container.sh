#!/bin/bash

# Check if Podman is available
if command -v podman &> /dev/null; then
    echo "podman"
    exit 0
fi

# Check if Docker is available
if command -v docker &> /dev/null; then
    echo "docker"
    exit 0
fi

# If neither Docker nor Podman is found, exit with code 1
echo "Neither Docker nor Podman is installed."
exit 1
