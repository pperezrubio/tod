# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

TOD (**T**he**O**ne**D**ev) is a CLI tool for OneDev 13+ that provides project management, CI/CD job execution, and an MCP server for AI tool integration. It's a single Go binary using Cobra for command routing.

## Architecture

### Core Files

| File | Purpose |
|------|---------|
| `main.go` | Entry point, all Cobra command definitions, flag registration, `init()` wiring |
| `config.go` | INI config loading from `$HOME/.todconfig` |
| `utils.go` | Shared helpers: `inferProject()`, `makeAPICall()`, `streamBuildLog()`, `getProjectId()`, ANSI color wrappers |

### Command Files

Each command lives in its own file following the pattern `{name}_command.go`:

| File | Commands | Project Resolution |
|------|----------|-------------------|
| `builds_command.go` | `builds` | `--project` / `-p` or git remote inference via `resolveProjectForBuilds()`. `--watch` / `-w` polls until terminal. `--interval` sets poll seconds. |
| `logs_command.go` | `logs <build-number>` | `--project` / `-p` or git remote inference via `resolveProjectForBuilds()` |
| `issues_command.go` | `issues [list|create|edit|close|comments|comment]` | `--project` / `-p` or git remote inference |
| `prs_command.go` | `prs [list|create|merge|approve|request-changes]` | `--project` / `-p` or git remote inference |
| `projects_command.go` | `projects` | N/A (lists all) |
| `agents_command.go` | `agents` | N/A (lists all) |
| `users_command.go` | `users` | N/A (lists all) |
| `secrets_command.go` | `secrets` | `--project` or git remote |
| `settings_command.go` | `settings` | `--project` or git remote |
| `webhooks_command.go` | `webhooks` | `--project` or git remote |
| `branches_command.go` | `branches [list|create|delete]` | `--project` or git remote |
| `iterations_command.go` | `iterations [list|create|close]` | `--project` or git remote |
| `artifacts_command.go` | `artifacts [list|download]` | `--project` + `--build` |
| `config_command.go` | `config [show|get|set|path]` | N/A |
| `create_project_command.go` | `create-project <name>` | N/A |
| `run_local_job_command.go` | `run-local <job-name>` | `--working-dir` or cwd |
| `run_job_command.go` | `run <job-name>` | `--branch` / `--tag` |
| `check_build_spec_command.go` | `check-build-spec` | `--working-dir` or cwd |
| `checkout_pull_request_command.go` | `checkout <pr-ref>` | `--working-dir` or cwd |
| `mcp_command.go` | `mcp` | N/A (MCP server mode) |

### Project Resolution Pattern

Most project-scoped commands use `--project` / `-p` flag with git remote inference fallback:

```go
// Defined in builds_command.go, shared by builds and logs
func resolveProjectForBuilds(cobraCmd *cobra.Command, logger *log.Logger) string {
    projectPath, _ := cobraCmd.Flags().GetString("project")
    if projectPath == "" {
        workingDir, _ := os.Getwd()
        _, inferredProject, err := inferProject(workingDir, logger)
        // ...
    }
    return projectPath
}
```

`inferProject()` in `utils.go` matches git remotes against OneDev's clone roots API to find the project path.

### Key Helpers (utils.go)

- `makeAPICall(req)` / `makeAPICallSimple(method, url, body)` — HTTP with Bearer auth
- `inferProject(workingDir, logger)` — git remote → OneDev project path
- `getProjectId(projectPath)` — project path → numeric ID
- `streamBuildLog(buildId, buildNumber, signalChannel)` — binary log protocol streaming
- `wrapWithRed/Green/Color/Bold()` — ANSI terminal formatting

## Development Commands

### Build
```bash
go build -o tod
```

### Test
```bash
go test ./...
```

### Run Examples
```bash
# List builds for current project (inferred from git remote)
./tod builds

# List builds with explicit project
./tod builds --project llm-proxy/tod

# Watch builds until all finish (polls every 10s, exits 0/1)
./tod builds --watch

# Watch with custom interval and project
./tod builds -w --interval 5 -p llm-proxy

# Stream build logs
./tod logs 109

# List issues
./tod issues --state open

# MCP server mode
./tod mcp --log-file /tmp/tod-mcp.log
```

## Configuration

### Config File: `$HOME/.todconfig` (INI format)

```ini
server-url=https://onedev.example.com
access-token=your-personal-access-token
```

### OneDev API Endpoints Used

| Endpoint | Purpose |
|----------|---------|
| `/~api/builds` | List/query builds |
| `/~api/issues` | List/query issues |
| `/~api/pulls` | List/query pull requests |
| `/~api/projects` | List projects |
| `/~api/projects/ids/{path}` | Resolve project path → ID |
| `/~api/streaming/build-logs/{id}` | Binary log streaming |
| `/~api/job-runs` | Run/cancel jobs |
| `/~api/mcp-helper/*` | MCP tool endpoints, clone roots, build spec check |
| `/~api/version/compatible-tod-versions` | Version compatibility |

## Key Implementation Details

### Binary Log Protocol
The streaming log endpoint uses a binary protocol (BigEndian int32 length prefix):
- Positive length → log entry JSON (styled message arrays with ANSI formatting)
- Negative length → build status string (SUCCESSFUL, FAILED, CANCELLED, TIMED_OUT)

### Build Queries
OneDev uses a query language for filtering: `"Project" is "my-project" and ("Job" is "Release")`. The `builds` command wraps user `--query` with project scoping automatically.

### Signal Handling
`run-local`, `run`, and `logs` commands handle Ctrl+C (SIGINT/SIGTERM) via goroutines to cancel running builds gracefully.

## Fork Details

This is a fork of [code.onedev.io/onedev/tod](https://code.onedev.io/onedev/tod) with extended CLI commands (issues, PRs, builds, branches, iterations, artifacts, agents, secrets, settings, users, webhooks, config management). Upstream only has `run-local`, `run`, `checkout`, `check-build-spec`, and `mcp`.
