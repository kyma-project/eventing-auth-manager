#!/usr/bin/env bash

service docker start
echo "Waiting for Docker to be up..." && sleep 30
set -e
curl -s https://raw.githubusercontent.com/k3d-io/k3d/main/install.sh | bash
k3d registry create registry.localhost --port 5001
k3d cluster create kcp --registry-use k3d-registry.localhost:5001
export KUBECONFIG=$(k3d kubeconfig write kcp)
TEST_EVENTING_AUTH_TARGET_KUBECONFIG_PATH=${KUBECONFIG} USE_EXISTING_CLUSTER=true make test-ci
k3d cluster delete kcp
