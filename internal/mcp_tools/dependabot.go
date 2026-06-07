package mcp_tools

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/dipsylala/ghas-mcp/api"
)

const (
	ListDependabotAlertsToolName = "list-dependabot-alerts"
	GetDependabotAlertToolName   = "get-dependabot-alert"
)

func init() {
	RegisterMCPTool(ListDependabotAlertsToolName, handleListDependabotAlerts)
	RegisterMCPTool(GetDependabotAlertToolName, handleGetDependabotAlert)
}

// ---- list-dependabot-alerts ----

type listDependabotAlertsRequest struct {
	Owner       string
	Repo        string // empty → org-level
	State       string // open | dismissed | fixed | auto_dismissed
	Severity    string // low | medium | high | critical
	Ecosystem   string // pip | npm | etc.
	PackageName string
	MaxResults  int // 0 = no user-imposed cap (server hard limit: 2000)
}

func parseListDependabotAlertsRequest(args map[string]interface{}) (*listDependabotAlertsRequest, error) {
	owner, err := extractRequiredString(args, "owner")
	if err != nil {
		return nil, err
	}
	repo, _ := extractOptionalString(args, "repo")
	state, _ := extractOptionalString(args, "state")
	severity, _ := extractOptionalString(args, "severity")
	ecosystem, _ := extractOptionalString(args, "ecosystem")
	pkg, _ := extractOptionalString(args, "package")

	maxResults := extractInt(args, "max_results", 0)
	return &listDependabotAlertsRequest{
		Owner:       owner,
		Repo:        repo,
		State:       state,
		Severity:    severity,
		Ecosystem:   ecosystem,
		PackageName: pkg,
		MaxResults:  maxResults,
	}, nil
}

func handleListDependabotAlerts(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	req, err := parseListDependabotAlertsRequest(args)
	if err != nil {
		return errorResult(err.Error()), nil
	}

	client, err := api.NewClient()
	if err != nil {
		return errorResult(fmt.Sprintf("authentication error: %v", err)), nil
	}

	params := url.Values{}
	if req.State != "" {
		params.Set("state", req.State)
	}
	if req.Severity != "" {
		params.Set("severity", req.Severity)
	}
	if req.Ecosystem != "" {
		params.Set("ecosystem", req.Ecosystem)
	}
	if req.PackageName != "" {
		params.Set("package", req.PackageName)
	}

	var path, scope string
	if req.Repo != "" {
		path = fmt.Sprintf("/repos/%s/%s/dependabot/alerts", req.Owner, req.Repo)
		scope = req.Owner + "/" + req.Repo
	} else {
		path = fmt.Sprintf("/orgs/%s/dependabot/alerts", req.Owner)
		scope = req.Owner + " (org)"
	}

	rawItems, err := client.GetAllPages(ctx, path, params)
	if err != nil {
		msg := fmt.Sprintf("GitHub API error: %v", err)
		if req.Repo == "" && strings.Contains(err.Error(), "404") {
			msg += " — org-level Dependabot alerts require a GitHub organisation. For personal accounts, supply a 'repo' parameter."
		}
		return errorResult(msg), nil
	}

	alerts := make([]api.DependabotAlert, 0, len(rawItems))
	for _, raw := range rawItems {
		var a api.DependabotAlert
		if err := json.Unmarshal(raw, &a); err == nil {
			alerts = append(alerts, a)
		}
	}

	return buildDependabotListResult(scope, req, alerts), nil
}

func buildDependabotListResult(scope string, req *listDependabotAlertsRequest, alerts []api.DependabotAlert) interface{} {
	type alertSummary struct {
		Number       int     `json:"number"`
		State        string  `json:"state"`
		Severity     string  `json:"severity"`
		Ecosystem    string  `json:"ecosystem"`
		Package      string  `json:"package"`
		Manifest     string  `json:"manifest_path,omitempty"`
		GHSAID       string  `json:"ghsa_id"`
		CVEID        string  `json:"cve_id,omitempty"`
		Summary      string  `json:"summary"`
		CVSSScore    float64 `json:"cvss_score,omitempty"`
		EPSSPct      float64 `json:"epss_percentage,omitempty"`
		FixedVersion string  `json:"fixed_version,omitempty"`
		HTMLURL      string  `json:"html_url"`
	}

	summaries := make([]alertSummary, 0, len(alerts))
	for _, a := range alerts {
		s := alertSummary{
			Number:    a.Number,
			State:     a.State,
			Severity:  a.SecurityAdvisory.Severity,
			Ecosystem: a.Dependency.Package.Ecosystem,
			Package:   a.Dependency.Package.Name,
			Manifest:  a.Dependency.ManifestPath,
			GHSAID:    a.SecurityAdvisory.GHSAID,
			CVEID:     a.SecurityAdvisory.CVEID,
			Summary:   a.SecurityAdvisory.Summary,
			HTMLURL:   a.HTMLURL,
		}
		if a.SecurityAdvisory.CVSS != nil {
			s.CVSSScore = a.SecurityAdvisory.CVSS.Score
		}
		if a.SecurityAdvisory.EPSS != nil {
			s.EPSSPct = a.SecurityAdvisory.EPSS.Percentage
		}
		if a.SecurityVulnerability.FirstPatchedVersion != nil {
			s.FixedVersion = a.SecurityVulnerability.FirstPatchedVersion.Identifier
		}
		summaries = append(summaries, s)
	}

	filters := map[string]string{}
	if req.State != "" {
		filters["state"] = req.State
	}
	if req.Severity != "" {
		filters["severity"] = req.Severity
	}
	if req.Ecosystem != "" {
		filters["ecosystem"] = req.Ecosystem
	}
	if req.PackageName != "" {
		filters["package"] = req.PackageName
	}

	// Apply max_results cap if set; also flag if the server pagination limit was hit.
	totalFetched := len(summaries)
	truncated := false
	if req.MaxResults > 0 && totalFetched > req.MaxResults {
		summaries = summaries[:req.MaxResults]
		truncated = true
	} else if totalFetched >= 2000 { // 2000 = maxPages(20) * perPage(100) in api/client.go
		truncated = true
	}

	result := map[string]interface{}{
		"scope":           scope,
		"total_count":     len(summaries),
		"filters_applied": filters,
		"alerts":          summaries,
	}
	if truncated {
		result["truncated"] = true
		result["total_fetched"] = totalFetched
		if req.MaxResults > 0 {
			result["truncation_reason"] = fmt.Sprintf("max_results cap of %d applied; %d total alerts were fetched", req.MaxResults, totalFetched)
		} else {
			result["truncation_reason"] = "server pagination limit of 2000 results reached; use filter parameters to narrow results"
		}
	}
	return result
}

// ---- get-dependabot-alert ----

func handleGetDependabotAlert(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	owner, err := extractRequiredString(args, "owner")
	if err != nil {
		return errorResult(err.Error()), nil
	}
	repo, err := extractRequiredString(args, "repo")
	if err != nil {
		return errorResult(err.Error()), nil
	}
	alertNumber := extractInt(args, "alert_number", 0)
	if alertNumber <= 0 {
		return errorResult("alert_number is required and must be a positive integer"), nil
	}

	client, err := api.NewClient()
	if err != nil {
		return errorResult(fmt.Sprintf("authentication error: %v", err)), nil
	}

	path := fmt.Sprintf("/repos/%s/%s/dependabot/alerts/%d", owner, repo, alertNumber)
	var alert api.DependabotAlert
	if err := client.GetJSON(ctx, path, nil, &alert); err != nil {
		return errorResult(fmt.Sprintf("GitHub API error: %v", err)), nil
	}

	return buildDependabotAlertDetail(owner+"/"+repo, &alert), nil
}

func buildDependabotAlertDetail(repo string, a *api.DependabotAlert) interface{} {
	cwes := make([]string, 0, len(a.SecurityAdvisory.CWEs))
	for _, cwe := range a.SecurityAdvisory.CWEs {
		cwes = append(cwes, fmt.Sprintf("%s (%s)", cwe.CWEID, cwe.Name))
	}

	advisory := map[string]interface{}{
		"ghsa_id":      a.SecurityAdvisory.GHSAID,
		"cve_id":       a.SecurityAdvisory.CVEID,
		"summary":      a.SecurityAdvisory.Summary,
		"description":  a.SecurityAdvisory.Description,
		"severity":     a.SecurityAdvisory.Severity,
		"cwes":         cwes,
		"published_at": a.SecurityAdvisory.PublishedAt,
	}
	if a.SecurityAdvisory.CVSS != nil {
		advisory["cvss_score"] = a.SecurityAdvisory.CVSS.Score
		advisory["cvss_vector"] = a.SecurityAdvisory.CVSS.VectorString
	}
	if a.SecurityAdvisory.EPSS != nil {
		advisory["epss_percentage"] = a.SecurityAdvisory.EPSS.Percentage
		advisory["epss_percentile"] = a.SecurityAdvisory.EPSS.Percentile
	}

	vuln := map[string]interface{}{
		"vulnerable_version_range": a.SecurityVulnerability.VulnerableVersionRange,
	}
	if a.SecurityVulnerability.FirstPatchedVersion != nil {
		vuln["first_patched_version"] = a.SecurityVulnerability.FirstPatchedVersion.Identifier
	}

	detail := map[string]interface{}{
		"repository": repo,
		"number":     a.Number,
		"state":      a.State,
		"severity":   a.Severity,
		"created_at": a.CreatedAt,
		"html_url":   a.HTMLURL,
		"dependency": map[string]interface{}{
			"ecosystem":     a.Dependency.Package.Ecosystem,
			"package":       a.Dependency.Package.Name,
			"manifest_path": a.Dependency.ManifestPath,
			"scope":         a.Dependency.Scope,
		},
		"security_advisory":      advisory,
		"security_vulnerability": vuln,
	}

	if a.State == "dismissed" {
		detail["dismissed_at"] = a.DismissedAt
		detail["dismissed_reason"] = a.DismissedReason
		detail["dismissed_comment"] = a.DismissedComment
		if a.DismissedBy != nil {
			detail["dismissed_by"] = a.DismissedBy.Login
		}
	}

	return detail
}
