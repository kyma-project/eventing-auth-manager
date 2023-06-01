#!/usr/bin/env bash

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # needs to be set if we want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked

service docker start
echo "Waiting for Docker to be up..." && sleep 30
set -e
curl -s https://raw.githubusercontent.com/k3d-io/k3d/main/install.sh | bash
k3d registry create registry.localhost --port 5000
k3d cluster create k3d --registry-use k3d-registry.localhost:5000
IMG="k3d-registry.localhost:5000/eventing-auth-manager:latest" make docker-build docker-push test
k3d cluster delete k3d