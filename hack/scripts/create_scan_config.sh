#!/usr/bin/env bash

# This script has the following arguments:
#                       filename of file to be created (mandatory)
#                       release tag (mandatory)
# ./create_scan_config image temp_scan_config.yaml tag      - use when bumping the config on the main branch

FILENAME=${1}
TAG=${2}

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # needs to be set if we want the ERR trap
set -o pipefail # prevents errors in a pipeline from being maskedPORT=5001

echo "Creating security scan configuration file:"

cat <<EOF | tee ${FILENAME}
module-name: eventing-auth-manager
kind: kcp
rc-tag: ${TAG}
bdba:
  - europe-docker.pkg.dev/kyma-project/prod/eventing-auth-manager:${TAG}
mend:
  language: golang-mod
  exclude:
    - "**/test/**"
    - "**/*_test.go"
checkmarx-one:
  preset: go-default
  exclude:
    - "**/test/**"
    - "**/*_test.go"
EOF
