# Orbital

Orbital is a lightweight Go tool that issues GitHub Actions self-hosted runner registration tokens via a GitHub App and uses them to start runner-compatible Docker containers on a user-provided Docker context.

It keeps the configured containers running, recreates them when they stop, and cleans them up gracefully when the process stops.

## Features

- **Docker context based operation**: Connects to the configured Docker context and starts runner-compatible containers there.
- **Dynamic runner registration**: Authenticates as a GitHub App and generates runner registration tokens on demand.
- **Runner container reconciliation**: Keeps the configured number of runner containers running and recreates them when necessary.
- **Graceful shutdown**: Stops and removes managed runner containers when Orbital receives a termination signal.

## Prerequisites

Before using Orbital, make sure you have:

- [Go](https://go.dev/) 1.26.1 or later
- [Docker](https://www.docker.com/)
- A reachable Docker context already configured via `docker context`
- A GitHub App installed on your organization with permission to manage self-hosted runners
- A runner image that can register itself using the environment variables passed by Orbital

The repository includes an example runner image implementation in:

- `example/androidrunner.Dockerfile`
- `example/entrypoint.sh`

That sample runner configures the GitHub Actions runner in ephemeral mode (`--ephemeral`) and expects the following environment variables:

- `GITHUB_URL`
- `RUNNER_NAME`
- `RUNNER_GROUP`
- `RUNNER_LABELS`
- `RUNNER_TOKEN`

## Configuration

Orbital uses a YAML configuration file.

Example:

```yaml
docker:
  context: default
  image: runner

github:
  org: ExampleOrg
  appId: 1234567
  installationId: 123456789
  pem: ~/.ssh/github-app.pem

runner:
  group: RunnerGroup
  labels:
    - linux
    - arm64
    - ubuntu24.04
  namePrefix: ubuntu
  count: 3

mount:
  source: /Volumes/cache
  target: /Volumes/cache

runtime:
  pollIntervalSeconds: 10
```

### Runner image contract

Orbital does not build, inspect, or validate the runner image specified by `docker.image`.
It starts the container with environment variables and expects the image itself to consume
those values and launch a GitHub Actions self-hosted runner.

Your `docker.image` must therefore be a runner-compatible image that:

- reads the environment variables passed by Orbital
- registers itself as a GitHub Actions self-hosted runner
- starts the runner process inside the container

Orbital passes the following environment variables to each runner container:

- `GITHUB_URL`
- `RUNNER_NAME`
- `RUNNER_GROUP`
- `RUNNER_LABELS`
- `RUNNER_TOKEN`

See `example/androidrunner.Dockerfile` and `example/entrypoint.sh` for a reference
implementation of a compatible runner image.

### Configuration fields

- `docker.context`: Name of the Docker context Orbital should use.
- `docker.image`: Docker image used for runner containers. This must be a runner-compatible image that consumes the environment variables passed by Orbital and starts a self-hosted runner. Orbital also derives managed container names from this value, so use a value that is safe to reuse in Docker container names.
- `github.org`: GitHub organization name.
- `github.appId`: GitHub App ID.
- `github.installationId`: GitHub App installation ID.
- `github.pem`: Path to the GitHub App private key (`.pem`).
- `runner.group`: Runner group name. Required by the sample runner image in `example/entrypoint.sh`.
- `runner.labels`: Custom labels assigned to the runner. Required by the sample runner image in `example/entrypoint.sh`.
- `runner.namePrefix`: Prefix used for runner names.
- `runner.count`: Number of runner containers Orbital should keep running.
- `mount.source`: Host path mounted into the runner container.
- `mount.target`: Destination path inside the runner container.
- `runtime.pollIntervalSeconds`: Polling interval, in seconds, for runner reconciliation.

### GitHub App permissions

Your GitHub App must be installed on the target organization and be able to create self-hosted runner registration tokens for that organization. In practice, configure the app with the permissions required for organization self-hosted runner administration before using Orbital.

## Quick start

Build the binary:

```bash
go build -o ./build/orbital ./app
```

Create a configuration file, for example:

```bash
cp ./example/config.yaml ./config.yaml
```

Then start Orbital:

```bash
./build/orbital start -c ./config.yaml
```

## What happens when it starts

When Orbital starts, it will:

1. Load the configuration file.
2. Check whether the configured Docker context is reachable.
3. Generate GitHub runner registration tokens through your GitHub App.
4. Start and maintain the configured number of runner containers.
5. Keep polling at the configured interval and recreate stopped containers if needed.
6. Stop and remove managed containers when the process is terminated, such as with `Ctrl+C`.

## Notes

- Managed container names are derived from `docker.image` and a 1-based index, for example `androidrunner-1`, `androidrunner-2`, and `androidrunner-3`.
- Runner names are generated from `runner.namePrefix` and a 1-based index, for example `ubuntu-1`, `ubuntu-2`, and `ubuntu-3`.
- Avoid including a trailing hyphen in `runner.namePrefix`, otherwise generated runner names become harder to read, such as `ubuntu--1`.
- If you use the sample runner image, `runner.group` and `runner.labels` must be set because the sample entrypoint validates both values before starting the runner.
- Orbital does not start container runtimes for you. Ensure the target Docker Engine behind the configured context is already running before starting Orbital.
