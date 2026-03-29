# Limarun

Limarun is a lightweight Go tool for managing GitHub Actions self-hosted runners as Docker containers inside a [Colima](https://github.com/abiosoft/colima) VM.

It ensures your runner containers are running, registers them dynamically through a GitHub App, and cleans them up gracefully when the process stops.

## Features

- **Automatic Colima startup**: Checks whether the configured Colima profile is running and starts it if needed.
- **Dynamic runner registration**: Authenticates as a GitHub App and generates runner registration tokens on demand.
- **Runner container reconciliation**: Keeps the configured number of runner containers running and recreates them when necessary.
- **Graceful shutdown**: Stops and removes managed runner containers when Limarun receives a termination signal.

## Prerequisites

Before using Limarun, make sure you have:

- [Go](https://go.dev/) 1.26.1 or later
- [Colima](https://github.com/abiosoft/colima)
- [Docker](https://www.docker.com/)
- A GitHub App installed on your organization with permission to manage self-hosted runners
- A runner image that can register itself using the environment variables passed by Limarun

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

Limarun uses a YAML configuration file.

Example:

```yaml
colima:
  profile: default

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
  image: androidrunner
  count: 3

mount:
  source: /Volumes/cache
  target: /Volumes/cache

runtime:
  pollIntervalSeconds: 10
```

### Configuration fields

- `colima.profile`: Name of the Colima profile to use.
- `github.org`: GitHub organization name.
- `github.appId`: GitHub App ID.
- `github.installationId`: GitHub App installation ID.
- `github.pem`: Path to the GitHub App private key (`.pem`).
- `runner.group`: Runner group name. Required by the sample runner image in `example/entrypoint.sh`.
- `runner.labels`: Custom labels assigned to the runner. Required by the sample runner image in `example/entrypoint.sh`.
- `runner.namePrefix`: Prefix used for runner names.
- `runner.image`: Docker image used for runner containers. Limarun also derives managed container names from this value, so use a value that is safe to reuse in Docker container names.
- `runner.count`: Number of runner containers Limarun should keep running.
- `mount.source`: Host path mounted into the runner container.
- `mount.target`: Destination path inside the runner container.
- `runtime.pollIntervalSeconds`: Polling interval, in seconds, for runner reconciliation.

### GitHub App permissions

Your GitHub App must be installed on the target organization and be able to create self-hosted runner registration tokens for that organization. In practice, configure the app with the permissions required for organization self-hosted runner administration before using Limarun.

## Quick start

Build the binary:

```bash
go build -o ./build/limarun ./app
```

Create a configuration file, for example:

```bash
cp ./example/config.yaml ./config.yaml
```

Then start Limarun:

```bash
./build/limarun start -c ./config.yaml
```

## What happens when it starts

When Limarun starts, it will:

1. Load the configuration file.
2. Check whether the configured Colima profile is running.
3. Start Colima automatically if it is not already running.
4. Generate GitHub runner registration tokens through your GitHub App.
5. Start and maintain the configured number of runner containers.
6. Keep polling at the configured interval and recreate stopped containers if needed.
7. Stop and remove managed containers when the process is terminated, such as with `Ctrl+C`.

## Notes

- Managed container names are derived from `runner.image` and a 1-based index, for example `androidrunner-1`, `androidrunner-2`, and `androidrunner-3`.
- Runner names are generated from `runner.namePrefix` and a 1-based index, for example `ubuntu-1`, `ubuntu-2`, and `ubuntu-3`.
- Avoid including a trailing hyphen in `runner.namePrefix`, otherwise generated runner names become harder to read, such as `ubuntu--1`.
- If you use the sample runner image, `runner.group` and `runner.labels` must be set because the sample entrypoint validates both values before starting the runner.
