#!/usr/bin/env bash

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # must be set if you want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked

# This script expects the following arguments:
# - RELEASE_TAG - release tag (required)
#
# ./check_publish_release_prerequisites.sh 1.0.0

export RELEASE_TAG=${1}
IMAGE_NAME="europe-docker.pkg.dev/kyma-project/prod/eventing-auth-manager"

# check if container image exists.
PROTOCOL=docker://
if [ $(skopeo list-tags ${PROTOCOL}${IMAGE_NAME} | jq '.Tags|any(. == env.RELEASE_TAG)') != "true" ]; then
    echo "Error: image do not exist: ${IMAGE_NAME}:${RELEASE_TAG}"
    exit 1
fi
echo "image ${IMAGE_NAME}:${RELEASE_TAG} exists"

