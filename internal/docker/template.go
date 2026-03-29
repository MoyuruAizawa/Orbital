package docker

const RunnerDockerfileTemplate = `FROM {{ .SourceImage }}

ARG RUNNER_TARGET_OS
ARG RUNNER_TARGET_ARCH
ARG RUNNER_VERSION=2.333.1

RUN set -eu; \
	if command -v apt-get >/dev/null 2>&1; then \
		export DEBIAN_FRONTEND=noninteractive; \
		apt-get update; \
		apt-get install -y --no-install-recommends bash curl tar gzip ca-certificates; \
		rm -rf /var/lib/apt/lists/*; \
	elif command -v apk >/dev/null 2>&1; then \
		apk add --no-cache bash curl tar gzip ca-certificates; \
	elif command -v dnf >/dev/null 2>&1; then \
		dnf install -y bash curl tar gzip ca-certificates; \
		dnf clean all; \
	elif command -v yum >/dev/null 2>&1; then \
		yum install -y bash curl tar gzip ca-certificates; \
		yum clean all; \
	else \
		echo "unsupported package manager: expected apt-get, apk, dnf, or yum" >&2; \
		exit 1; \
	fi; \
	echo "resolved runner target platform: ${RUNNER_TARGET_OS}/${RUNNER_TARGET_ARCH}"; \
	case "${RUNNER_TARGET_OS}/${RUNNER_TARGET_ARCH}" in \
		linux/amd64) runner_arch="x64" ;; \
		linux/arm64) runner_arch="arm64" ;; \
		*) echo "unsupported runner platform: ${RUNNER_TARGET_OS}/${RUNNER_TARGET_ARCH}" >&2; exit 1 ;; \
	esac; \
	mkdir -p /actions-runner; \
	curl -L --fail --show-error -o /tmp/actions-runner.tar.gz \
		"https://github.com/actions/runner/releases/download/v${RUNNER_VERSION}/actions-runner-${RUNNER_TARGET_OS}-${runner_arch}-${RUNNER_VERSION}.tar.gz"; \
	tar xzf /tmp/actions-runner.tar.gz -C /actions-runner; \
	rm -f /tmp/actions-runner.tar.gz

ENV RUNNER_ALLOW_RUNASROOT=1
ENV RUNNER_WORKDIR="_work"

WORKDIR /actions-runner

COPY entrypoint.sh /entrypoint.sh

RUN chmod +x /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]
`

const RunnerEntrypoint = `#!/usr/bin/env bash
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

cleanup() {
  ./config.sh remove --token "${RUNNER_TOKEN}" --unattended || true
  rm -f .runner .credentials .credentials_rsaparams || true
}

trap cleanup EXIT

config_args=(
  --url "${GITHUB_URL}" \
  --token "${RUNNER_TOKEN}" \
  --name "${RUNNER_NAME}" \
  --work "${RUNNER_WORKDIR}" \
  --ephemeral \
  --unattended \
  --replace
)

if [[ -n "${RUNNER_GROUP:-}" ]]; then
  config_args+=(--runnergroup "${RUNNER_GROUP}")
fi

if [[ -n "${RUNNER_LABELS:-}" ]]; then
  config_args+=(--labels "${RUNNER_LABELS}")
fi

./config.sh "${config_args[@]}"

exec ./run.sh
`
