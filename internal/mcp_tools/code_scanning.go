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
	ListCodeScanningAlertsToolName = "list-code-scanning-alerts"
	GetCodeScanningAlertToolName   = "get-code-scanning-alert"
)

func init() {
	RegisterMCPTool(ListCodeScanningAlertsToolName, handleListCodeScanningAlerts)
	RegisterMCPTool(GetCodeScanningAlertToolName, handleGetCodeScanningAlert)
}

// ---- list-code-scanning-alerts ----

type listCodeScanningAlertsRequest struct {
	Owner    string
	Repo     string // empty → org-level
	State    string // open | dismissed | fixed (default: open)
	Severity string // critical | high | medium | low | warning | note | error
	ToolName string
	Ref      string
}

func parseListCodeScanningAlertsRequest(args map[string]interface{}) (*listCodeScanningAlertsRequest, error) {
	owner, err := extractRequiredString(args, "owner")
	if err != nil {
		return nil, err
	}
	repo, _ := extractOptionalString(args, "repo")
	state, _ := extractOptionalString(args, "state")
	severity, _ := extractOptionalString(args, "severity")
	toolName, _ := extractOptionalString(args, "tool_name")
	ref, _ := extractOptionalString(args, "ref")

	return &listCodeScanningAlertsRequest{
		Owner:    owner,
		Repo:     repo,
		State:    state,
		Severity: severity,
		ToolName: toolName,
		Ref:      ref,
	}, nil
}

func handleListCodeScanningAlerts(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	req, err := parseListCodeScanningAlertsRequest(args)
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
	if req.ToolName != "" {
		params.Set("tool_name", req.ToolName)
	}
	if req.Ref != "" {
		params.Set("ref", req.Ref)
	}

	var path string
	var scope string
	if req.Repo != "" {
		path = fmt.Sprintf("/repos/%s/%s/code-scanning/alerts", req.Owner, req.Repo)
		scope = req.Owner + "/" + req.Repo
	} else {
		path = fmt.Sprintf("/orgs/%s/code-scanning/alerts", req.Owner)
		scope = req.Owner + " (org)"
	}

	rawItems, err := client.GetAllPages(ctx, path, params)
	if err != nil {
		msg := fmt.Sprintf("GitHub API error: %v", err)
		if req.Repo == "" && strings.Contains(err.Error(), "404") {
			msg += " — org-level code scanning alerts require a GitHub organisation. For personal accounts, supply a 'repo' parameter."
		}
		return errorResult(msg), nil
	}

	alerts := make([]api.CodeScanningAlert, 0, len(rawItems))
	for _, raw := range rawItems {
		var a api.CodeScanningAlert
		if err := json.Unmarshal(raw, &a); err == nil {
			alerts = append(alerts, a)
		}
	}

	return buildCodeScanningListResult(scope, req, alerts), nil
}

func buildCodeScanningListResult(scope string, req *listCodeScanningAlertsRequest, alerts []api.CodeScanningAlert) interface{} {
	type alertSummary struct {
		Number   int    `json:"number"`
		State    string `json:"state"`
		Severity string `json:"severity"`
		RuleID   string `json:"rule_id"`
		RuleName string `json:"rule_name"`
		Tool     string `json:"tool"`
		File     string `json:"file,omitempty"`
		Line     int    `json:"start_line,omitempty"`
		Ref      string `json:"ref,omitempty"`
		Message  string `json:"message,omitempty"`
		HTMLURL  string `json:"html_url"`
	}

	summaries := make([]alertSummary, 0, len(alerts))
	for _, a := range alerts {
		s := alertSummary{
			Number:   a.Number,
			State:    a.State,
			Severity: severityLabel(a.Rule.SecuritySeverityLevel, a.Rule.Severity),
			RuleID:   a.Rule.ID,
			RuleName: a.Rule.Name,
			Tool:     a.Tool.Name,
			HTMLURL:  a.HTMLURL,
		}
		if a.MostRecentInstance != nil {
			s.Ref = a.MostRecentInstance.Ref
			if a.MostRecentInstance.Location != nil {
				s.File = a.MostRecentInstance.Location.Path
				s.Line = a.MostRecentInstance.Location.StartLine
			}
			if a.MostRecentInstance.Message != nil {
				s.Message = a.MostRecentInstance.Message.Text
			}
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
	if req.ToolName != "" {
		filters["tool_name"] = req.ToolName
	}
	if req.Ref != "" {
		filters["ref"] = req.Ref
	}

	return map[string]interface{}{
		"scope":           scope,
		"total_count":     len(summaries),
		"filters_applied": filters,
		"alerts":          summaries,
	}
}

// ---- get-code-scanning-alert ----

func handleGetCodeScanningAlert(ctx context.Context, args map[string]interface{}) (interface{}, error) {
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

	path := fmt.Sprintf("/repos/%s/%s/code-scanning/alerts/%d", owner, repo, alertNumber)
	var alert api.CodeScanningAlert
	if err := client.GetJSON(ctx, path, nil, &alert); err != nil {
		return errorResult(fmt.Sprintf("GitHub API error: %v", err)), nil
	}

	return buildCodeScanningAlertDetail(owner+"/"+repo, &alert), nil
}

func buildCodeScanningAlertDetail(repo string, a *api.CodeScanningAlert) interface{} {
	detail := map[string]interface{}{
		"repository": repo,
		"number":     a.Number,
		"state":      a.State,
		"created_at": a.CreatedAt,
		"html_url":   a.HTMLURL,
		"rule": map[string]interface{}{
			"id":                a.Rule.ID,
			"name":              a.Rule.Name,
			"severity":          a.Rule.Severity,
			"security_severity": a.Rule.SecuritySeverityLevel,
			"description":       a.Rule.Description,
			"full_description":  a.Rule.FullDescription,
			"tags":              a.Rule.Tags,
		},
		"tool": map[string]interface{}{
			"name":    a.Tool.Name,
			"version": a.Tool.Version,
		},
	}

	if a.State == "dismissed" {
		detail["dismissed_at"] = a.DismissedAt
		detail["dismissed_reason"] = a.DismissedReason
		if a.DismissedBy != nil {
			detail["dismissed_by"] = a.DismissedBy.Login
		}
	}

	if a.MostRecentInstance != nil {
		inst := map[string]interface{}{
			"ref":   a.MostRecentInstance.Ref,
			"state": a.MostRecentInstance.State,
		}
		if a.MostRecentInstance.Location != nil {
			inst["file"] = a.MostRecentInstance.Location.Path
			inst["start_line"] = a.MostRecentInstance.Location.StartLine
			inst["end_line"] = a.MostRecentInstance.Location.EndLine
		}
		if a.MostRecentInstance.Message != nil {
			inst["message"] = a.MostRecentInstance.Message.Text
		}
		detail["most_recent_instance"] = inst
	}

	return detail
}

// severityLabel returns a unified severity label, preferring security_severity_level.
func severityLabel(secLevel, ruleSev string) string {
	if secLevel != "" {
		return secLevel
	}
	return ruleSev
}

// errorResult wraps an error string as a map so it renders cleanly in the LLM.
func errorResult(msg string) interface{} {
	return map[string]interface{}{"error": msg}
}
