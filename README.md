# ghas-mcp

A [Model Context Protocol](https://modelcontextprotocol.io) (MCP) server that exposes **GitHub Advanced Security** alerts to AI assistants. Written in Go - ships as a single static binary with no runtime dependencies.

Ask your AI assistant things like:

- *"List all critical Dependabot alerts in my-org/api-service"*
- *"What CWE is code scanning alert #42 in my-org/api-service? Which file is it in?"*
- *"Are there any actively valid leaked secrets in the my-org organisation?"*
- *"Which npm packages across my org have critical CVEs and no fix available?"*

## Tools

| Tool | Scope | Description |
| --- | --- | --- |
| `list-code-scanning-alerts` | repo or org | List alerts, filter by state / severity / tool / ref |
| `get-code-scanning-alert` | repo | Full alert: rule description, CWE tags, file + line, dismissal history |
| `list-dependabot-alerts` | repo or org | List alerts with CVSS score, EPSS %, ecosystem, first patched version |
| `get-dependabot-alert` | repo | Full advisory: description, CVSS vector, EPSS, CWEs, version range |
| `list-secret-scanning-alerts` | repo or org | List by state / secret type - **actual secret values are never returned** |

All tools are **read-only**.

## Prerequisites

- A GitHub account with GHAS features enabled on your repositories or organisation
- A GitHub personal access token (see [Authentication](#authentication))

## Installation

### Option 1 - go install

```sh
go install github.com/dipsylala/ghas-mcp@latest
```

### Option 2 - release binary

Download the binary for your platform from [Releases](https://github.com/dipsylala/ghas-mcp/releases) and place it somewhere on your `PATH`.

| Platform | File |
| --- | --- |
| Windows (amd64) | `ghas-mcp-windows-amd64.exe` |
| Windows (arm64) | `ghas-mcp-windows-arm64.exe` |
| macOS (Apple Silicon) | `ghas-mcp-darwin-arm64` |
| macOS (Intel) | `ghas-mcp-darwin-amd64` |
| Linux (amd64) | `ghas-mcp-linux-amd64` |
| Linux (arm64) | `ghas-mcp-linux-arm64` |

### Option 3 - build from source

```sh
git clone https://github.com/dipsylala/ghas-mcp
cd ghas-mcp
go build -o ghas-mcp .
```

## Authentication

The server resolves a GitHub token in this order:

1. `GITHUB_TOKEN` environment variable
2. `gh auth token` - the active gh CLI session (run `gh auth login` once to set it up; works on all platforms including Windows)

### Token scopes required

| Alert type | Classic PAT | Fine-grained PAT |
| --- | --- | --- |
| Code scanning | `security_events` | Code scanning alerts: **Read** |
| Dependabot | `security_events` | Dependabot alerts: **Read** |
| Secret scanning | `security_events` | Secret scanning alerts: **Read** |

A single classic PAT with `security_events` covers all three alert types. Fine-grained PATs require each permission individually and must be scoped to the relevant repositories or organisation.

> **Note:** Dependabot alerts also require the repository to have the Dependency graph enabled (it is on by default for public repos).

## MCP client configuration

### VS Code - `.vscode/mcp.json`

```json
{
  "servers": {
    "ghas": {
      "type": "stdio",
      "command": "ghas-mcp",
      "env": {
        "GITHUB_TOKEN": "${env:GITHUB_TOKEN}"
      }
    }
  }
}
```

> **Warning:** Do not paste a raw token value (`ghp_...`) into `.vscode/mcp.json`. Keep the `${env:GITHUB_TOKEN}` reference and set `GITHUB_TOKEN` in your shell or system environment instead. Alternatively, omit the `env` block entirely and use `gh auth login`. `.vscode/mcp.json` is in `.gitignore` - do not force-add it to git.

### Claude Desktop - `claude_desktop_config.json`

The recommended approach is to rely on the `gh` CLI - run `gh auth login` once and omit the `env` block entirely:

```json
{
  "mcpServers": {
    "ghas": {
      "command": "ghas-mcp"
    }
  }
}
```

If you need to pass a token explicitly, set `GITHUB_TOKEN` in your shell profile (`.bashrc`, `.zshrc`, Windows system environment variables, etc.) before launching Claude Desktop. Claude Desktop does not support variable interpolation in its config, so the token must already be present in the process environment - not embedded in the JSON file.

> **Warning:** Do not paste a raw token value (`ghp_...`) directly into `claude_desktop_config.json`. It is a plaintext file that may be synced by cloud backup tools (iCloud, Dropbox, etc.).

### Cursor / other clients

Any MCP-compatible client that supports `stdio` transport works. Set `command` to the binary path and pass the token via the `GITHUB_TOKEN` environment variable.

## Flags

| Flag | Description |
| --- | --- |
| `--version` | Print version and exit |
| `--verbose` | Write logs to stderr |
| `--log <path>` | Write logs to a file instead of stderr |

Logging is disabled by default so it does not interfere with the stdio JSON-RPC transport. Enable it with `--verbose` or `--log` when debugging.

## Example prompts

```text
Show me all open code scanning alerts in my-org/api-service
```

```text
What is the full detail of code scanning alert #12 in my-org/api-service?
Include the CWE, the file, and whether it has been dismissed.
```

```text
List all critical Dependabot alerts across the my-org organisation.
Group them by ecosystem.
```

```text
Is the lodash vulnerability in my-org/frontend fixed yet?
What version should I upgrade to?
```

```text
Are there any open secret scanning alerts in my-org/backend that
involve tokens which are still active?
```

## Building release binaries

```powershell
.\build.ps1 -Version 1.0.0
```

Produces binaries for all five platforms in `dist/`.

## Why MCP rather than agent skills?

An alternative approach would be a set of agent skills that instruct the LLM to call `gh api` directly. That works for ad-hoc use, but has meaningful drawbacks for security-adjacent tooling:

| | `gh api` + skill | ghas-mcp |
| --- | --- | --- |
| **Input validation** | LLM constructs the URL and params from memory — error-prone | JSON schema enforced at the protocol level before any API call |
| **Output normalisation** | Raw GitHub API fields (e.g. `security_severity_level`) reach the LLM as-is | Fields are renamed, filtered, and normalised to consistent names |
| **Pagination** | LLM must remember to pass `--paginate`; no result cap | Automatic, with a 2000-item hard cap and `truncated` flag when hit |
| **Client compatibility** | Requires shell access — won't work in Claude Desktop without a terminal | Works in any MCP-compatible client regardless of shell access |
| **Token handling** | `gh auth login` handles credentials; no env var required | Same — `gh auth login` works with no env var needed. The token is read once at process start and never passed through LLM-generated strings |
| **Determinism** | Non-deterministic — the LLM reconstructs the call each time | Deterministic — the tool schema defines exactly what can be called and how |

The `gh` CLI is a good option for general GitHub tasks (issues, PRs, releases). For GHAS specifically — where the field names are non-obvious, pagination is important, and the data is security-sensitive — a typed MCP tool is the more reliable and auditable choice.

## Security notes

- Secret values are **never** returned by any tool - only metadata (type, state, validity flag).
- The server is read-only. It cannot dismiss, reopen, or modify any alert.
- Token is read once at startup from the environment; it is never logged.
- **Never hardcode a token** in an MCP client config file. For VS Code, use the `${env:GITHUB_TOKEN}` reference (VS Code resolves this on all platforms). For other clients, set `GITHUB_TOKEN` in the system/shell environment before launching, or use `gh auth login`.
- `.vscode/mcp.json` is in `.gitignore`. Do not force-add it to version control with a token value inside.
- **stdio transport only.** The server deliberately does not expose an HTTP/SSE listener. stdio means the token stays in the process environment, no network port is opened, and no additional authentication layer is needed for the MCP endpoint itself. If you need remote/multi-client access, you would also need to secure the MCP endpoint - that is out of scope for this tool.

## License

MIT
