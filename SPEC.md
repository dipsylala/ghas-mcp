# ghas-mcp — Tool Specification

This document is the authoritative reference for AI assistants using the ghas-mcp MCP server.
It describes every tool, its parameters, response fields, known constraints, and recommended usage patterns.

---

## Overview

ghas-mcp exposes five read-only tools covering three GitHub Advanced Security alert surfaces:

| Category | Tools |
|---|---|
| Code scanning | `list-code-scanning-alerts`, `get-code-scanning-alert` |
| Dependabot | `list-dependabot-alerts`, `get-dependabot-alert` |
| Secret scanning | `list-secret-scanning-alerts` |

### General rules

- **Scope**: Supply `owner` + `repo` for repository-level queries. Supply only `owner` for org-level queries.
- **Pagination**: All list tools automatically page through the full result set (up to 2000 items). There is no need to request pages manually.
- **Authentication**: The server resolves `GITHUB_TOKEN` from the environment, then falls back to running `gh auth token` (the active gh CLI session). If neither is present, tools return an `error` field.
- **Personal accounts vs organisations**: Org-level queries (omitting `repo`) use the `/orgs/{owner}/...` GitHub API endpoint, which only works for GitHub organisations. For personal accounts, always supply a `repo` parameter.
- **Secret values**: Secret scanning tools never return the actual secret string — only metadata.
- **Write operations**: No tool modifies any alert. Dismissal or state changes must be done via the GitHub UI or API.

---

## Tool reference

### `list-code-scanning-alerts`

Returns a list of code scanning alerts for a repository or organisation, with one summary record per alert.

#### Parameters

| Name | Type | Required | Allowed values | Description |
|---|---|---|---|---|
| `owner` | string | yes | — | GitHub org or user name |
| `repo` | string | no | — | Repository name (without owner). Omit for org scope |
| `state` | string | no | `open` `dismissed` `fixed` | Filter by alert state (default: all) |
| `severity` | string | no | `critical` `high` `medium` `low` `warning` `note` `error` | Filter by security severity |
| `tool_name` | string | no | — | Filter by analysis tool (e.g. `CodeQL`, `Semgrep`) |
| `ref` | string | no | — | Git ref (e.g. `refs/heads/main`) |

#### Response fields

```jsonc
{
  "scope": "my-org/api-service",       // owner/repo or owner (org)
  "total_count": 3,
  "filters_applied": { "state": "open", "severity": "high" },
  "alerts": [
    {
      "number": 42,
      "state": "open",                 // open | dismissed | fixed
      "severity": "high",              // security_severity_level when present, else rule severity
      "rule_id": "java/sql-injection",
      "rule_name": "SQL injection",
      "tool": "CodeQL",
      "file": "src/main/java/App.java",
      "start_line": 88,
      "ref": "refs/heads/main",
      "message": "This query depends on a user-provided value.",
      "html_url": "https://github.com/my-org/api-service/security/code-scanning/42"
    }
  ]
}
```

#### Notes

- `severity` uses `security_severity_level` when available (critical/high/medium/low). For tools that only emit rule severity (warning/note/error), the rule severity is used instead.
- Org-level queries require the token to have org-wide `security_events` access.

---

### `get-code-scanning-alert`

Returns full details for a single code scanning alert, including the rule's complete description, all CWE tags, tool version, and exact code location.

#### Parameters

| Name | Type | Required | Description |
|---|---|---|---|
| `owner` | string | yes | GitHub org or user name |
| `repo` | string | yes | Repository name (without owner) |
| `alert_number` | integer | yes | Alert number from `list-code-scanning-alerts` or the GitHub URL |

#### Response fields

```jsonc
{
  "repository": "my-org/api-service",
  "number": 42,
  "state": "open",
  "created_at": "2024-11-01T09:12:00Z",
  "html_url": "https://github.com/...",
  "rule": {
    "id": "java/sql-injection",
    "name": "SQL injection",
    "severity": "error",
    "security_severity": "high",
    "description": "Building a SQL query from user-controlled sources...",
    "full_description": "...",
    "tags": ["security", "external/cwe/cwe-089", "external/cwe/cwe-564"]
  },
  "tool": {
    "name": "CodeQL",
    "version": "2.16.3"
  },
  // Present when state == "dismissed":
  "dismissed_at": "2024-11-15T14:00:00Z",
  "dismissed_reason": "false positive",
  "dismissed_by": "alice",
  // Most recent occurrence:
  "most_recent_instance": {
    "ref": "refs/heads/main",
    "state": "open",
    "file": "src/main/java/App.java",
    "start_line": 88,
    "end_line": 88,
    "message": "This query depends on a user-provided value."
  }
}
```

#### Notes

- CWE identifiers appear in `rule.tags` with the prefix `external/cwe/cwe-` (e.g. `external/cwe/cwe-089` = CWE-89 SQL Injection).
- `dismissed_at`, `dismissed_reason`, and `dismissed_by` are only present when `state == "dismissed"`.

---

### `list-dependabot-alerts`

Returns a list of Dependabot vulnerability alerts for a repository or organisation.

#### Parameters

| Name | Type | Required | Allowed values | Description |
|---|---|---|---|---|
| `owner` | string | yes | — | GitHub org or user name |
| `repo` | string | no | — | Repository name. Omit for org scope |
| `state` | string | no | `open` `dismissed` `fixed` `auto_dismissed` | Filter by state (default: all) |
| `severity` | string | no | `low` `medium` `high` `critical` | Filter by severity |
| `ecosystem` | string | no | — | Package ecosystem: `npm` `pip` `maven` `rubygems` `nuget` `cargo` `composer` `go` `rust` `pub` |
| `package` | string | no | — | Exact package name (e.g. `lodash`) |

#### Response fields

```jsonc
{
  "scope": "my-org",
  "total_count": 14,
  "filters_applied": { "severity": "critical", "ecosystem": "npm" },
  "alerts": [
    {
      "number": 7,
      "state": "open",
      "severity": "critical",
      "ecosystem": "npm",
      "package": "lodash",
      "manifest_path": "package.json",
      "ghsa_id": "GHSA-35jh-r3h4-6jhm",
      "cve_id": "CVE-2019-10744",
      "summary": "Prototype Pollution in lodash",
      "cvss_score": 9.8,
      "epss_percentage": 0.97,    // 0–100 scale; high = more likely exploited in the wild
      "fixed_version": "4.17.21",
      "html_url": "https://github.com/my-org/..."
    }
  ]
}
```

#### Notes

- `fixed_version` is the first version that contains a patch. It is absent if no patch exists yet.
- `epss_percentage` is the EPSS probability (0–100 %). Values above ~50 indicate a higher-than-average exploitation likelihood.
- CVSS scores are advisory-level; the same vulnerability may score differently across different databases.

---

### `get-dependabot-alert`

Returns the full advisory details for a single Dependabot alert, including the complete vulnerability description, CVSS vector string, EPSS percentile, all CWE identifiers, and the vulnerable version range.

#### Parameters

| Name | Type | Required | Description |
|---|---|---|---|
| `owner` | string | yes | GitHub org or user name |
| `repo` | string | yes | Repository name (without owner) |
| `alert_number` | integer | yes | Alert number from `list-dependabot-alerts` or the GitHub URL |

#### Response fields

```jsonc
{
  "repository": "my-org/api-service",
  "number": 7,
  "state": "open",
  "severity": "critical",
  "created_at": "2024-10-20T08:00:00Z",
  "html_url": "https://github.com/...",
  "dependency": {
    "ecosystem": "npm",
    "package": "lodash",
    "manifest_path": "package.json",
    "scope": "runtime"            // runtime | development
  },
  "security_advisory": {
    "ghsa_id": "GHSA-35jh-r3h4-6jhm",
    "cve_id": "CVE-2019-10744",
    "summary": "Prototype Pollution in lodash",
    "description": "Versions of lodash prior to 4.17.12 are vulnerable...",
    "severity": "critical",
    "cvss_score": 9.8,
    "cvss_vector": "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H",
    "epss_percentage": 0.97,
    "epss_percentile": 99.1,
    "cwes": ["CWE-1321 (Improperly Controlled Modification of Object Prototype Attributes)"],
    "published_at": "2019-07-26T00:00:00Z"
  },
  "security_vulnerability": {
    "vulnerable_version_range": "< 4.17.21",
    "first_patched_version": "4.17.21"
  },
  // Present when state == "dismissed":
  "dismissed_at": "2024-11-01T12:00:00Z",
  "dismissed_reason": "tolerable_risk",
  "dismissed_comment": "Internal tool only, not exposed",
  "dismissed_by": "bob"
}
```

---

### `list-secret-scanning-alerts`

Returns metadata for secret scanning alerts. **The actual secret value is never included in any response.**

#### Parameters

| Name | Type | Required | Allowed values | Description |
|---|---|---|---|---|
| `owner` | string | yes | — | GitHub org or user name |
| `repo` | string | no | — | Repository name. Omit for org scope |
| `state` | string | no | `open` `resolved` | Filter by state (default: all) |
| `secret_type` | string | no | — | Secret type slug (see [GitHub docs](https://docs.github.com/en/code-security/secret-scanning/introduction/supported-secret-scanning-patterns)) |

#### Response fields

```jsonc
{
  "scope": "my-org",
  "total_count": 5,
  "filters_applied": { "state": "open" },
  "alerts": [
    {
      "number": 3,
      "state": "open",
      "secret_type": "github_personal_access_token",
      "secret_type_display_name": "GitHub Personal Access Token",
      "validity": "active",           // active | inactive | unknown
      "resolution": null,             // present if resolved
      "publicly_leaked": false,       // true if found in a public commit
      "multi_repo": false,            // true if found across multiple repos
      "push_protection_bypassed": true,
      "created_at": "2024-11-01T10:00:00Z",
      "html_url": "https://github.com/..."
    }
  ]
}
```

#### Notes

- `validity` reflects whether GitHub has verified that the token is still active with the issuing service. `active` means the secret is live and poses immediate risk.
- `publicly_leaked` is a strong indicator of priority — the secret has been exposed to the internet.
- `push_protection_bypassed` means the committer explicitly overrode a push protection block to commit the secret.
- To look up valid `secret_type` slugs, see [GitHub's supported patterns list](https://docs.github.com/en/code-security/secret-scanning/introduction/supported-secret-scanning-patterns).

---

## Recommended workflows

### Triage a repository's security posture

1. Call `list-code-scanning-alerts` with `owner` + `repo` + `state=open` to get all active SAST findings.
2. Call `list-dependabot-alerts` with `owner` + `repo` + `state=open` + `severity=critical` for the highest-risk dependency issues.
3. Call `list-secret-scanning-alerts` with `owner` + `repo` + `state=open` to check for exposed credentials.

### Investigate a specific code scanning alert

1. Call `list-code-scanning-alerts` to find the alert number.
2. Call `get-code-scanning-alert` with that number to get the full rule description, CWE tags, and exact file/line.
3. Use the CWE identifiers in `rule.tags` (format: `external/cwe/cwe-NNN`) to look up remediation guidance.

### Assess a dependency vulnerability

1. Call `list-dependabot-alerts` filtered by `package` to find the alert number.
2. Call `get-dependabot-alert` to get the full advisory, including the exact vulnerable version range, the first patched version, and the CVSS vector.
3. If `first_patched_version` is present, the fix is to upgrade to that version or higher.

### Prioritise secrets by risk

When triaging secret scanning alerts, prioritise by:
1. `validity == "active"` — the secret is confirmed live
2. `publicly_leaked == true` — the secret has been public
3. `push_protection_bypassed == true` — bypassed a control, indicating intent
4. `multi_repo == true` — the same secret appears across multiple repositories

---

## Error handling

When a tool cannot complete, it returns a JSON object with a single `error` field:

```json
{ "error": "GitHub API error 403: Resource not accessible by personal access token" }
```

Common errors:

| Error pattern | Likely cause |
|---|---|
| `token not found` | `GITHUB_TOKEN` is unset and no gh CLI session exists |
| `GitHub API error 401` | Token is invalid or expired |
| `GitHub API error 403` | Token lacks the required scope for this alert type |
| `GitHub API error 404` (repo query) | Repository does not exist or is not accessible to the token |
| `GitHub API error 404` (org query) | The `owner` is a personal account, not a GitHub organisation — supply a `repo` parameter |
| `GitHub API error 422` | Invalid filter value (check allowed values for `state`, `severity`, etc.) |

---

## Data freshness

All data is fetched live from the GitHub REST API at the time of the tool call. There is no local cache. Results reflect the current state of alerts on GitHub.
