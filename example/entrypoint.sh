o#!/usr/bin/env bash
set -euo pipefail

cd /actions-runner

if [[ -z "${GITHUB_URL:-}" ]]; then
  echo "GITHUB_URL is required" >&2
  exit 1
fi

if [[ -z "${RUNNER_TOKEN:-}" ]]; then
  echo "RUNNER_TOKEN is required" >&2
  exit 1
fi

if [[ -z "${RUNNER_NAME:-}" ]]; then
  echo "RUNNER_NAME is required" >&2
  exit 1
fi

if [[ -z "${RUNNER_GROUP:-}" ]]; then
  echo "RUNNER_GROUP is required" >&2
  exit 1
fi

if [[ -z "${RUNNER_LABELS:-}" ]]; then
  echo "RUNNER_LABELS is required" >&2
  exit 1
fi

./config.sh \
  --url "${GITHUB_URL}" \
  --token "${RUNNER_TOKEN}" \
  --name "${RUNNER_NAME}" \
  --runnergroup "${RUNNER_GROUP}" \
  --labels "${RUNNER_LABELS}" \
  --work "${RUNNER_WORKDIR}" \
  --ephemeral \
  --unattended \
  --replace

cleanup() {
  rm -f .runner .credentials .credentials_rsaparams || true
}

trap cleanup EXIT

exec ./run.sh