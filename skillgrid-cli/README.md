# skillgrid-cli

Go CLI that installs and configures AI agent tooling.

Requires **Go 1.22+**; `go.mod` pins `toolchain go1.24.0` for `modernc.org/sqlite` (Taskfile uses `GOTOOLCHAIN=auto` to fetch it if needed).

## Build

```
task build
```

## Test

```
task test-cli
```

## Usage

```
skillgrid-cli install --dry-run
skillgrid-cli sync-repo --sync-repo /extra/path
```

Validation logs are written to `~/.skillgrid/logs/install.log`.
