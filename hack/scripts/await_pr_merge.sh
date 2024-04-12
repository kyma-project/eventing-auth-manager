#!/usr/bin/env bash

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # needs to be set if we want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked

# Expected variables:
# EVENTING_AUTH_MANAGER_REPO - Kyma repository
# PR_NUMBER - Number of the PR with the changes to be merged

# wait until the PR is merged.
until [ $(gh pr view ${PR_NUMBER} --json mergedAt | jq -r '.mergedAt') != "null" ]; do
  echo "Waiting for https://github.com/${EVENTING_AUTH_MANAGER_REPO}/pull/${PR_NUMBER} to be merged"
  sleep 5
done

echo "The PR: ${PR_NUMBER} was merged at: $(gh pr view ${PR_NUMBER} --json mergedAt | jq -r '.mergedAt')"
