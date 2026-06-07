package mcp_tools

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/dipsylala/ghas-mcp/api"
)

const ListSecretScanningAlertsToolName = "list-secret-scanning-alerts"

func init() {
	RegisterMCPTool(ListSecretScanningAlertsToolName, handleListSecretScanningAlerts)
}

type listSecretScanningAlertsRequest struct {
	Owner      string
	Repo       string // empty → org-level
	State      string // open | resolved
	SecretType string // e.g. "mailchimp_api_key"
}

func parseListSecretScanningAlertsRequest(args map[string]interface{}) (*listSecretScanningAlertsRequest, error) {
	owner, err := extractRequiredString(args, "owner")
	if err != nil {
		return nil, err
	}
	repo, _ := extractOptionalString(args, "repo")
	state, _ := extractOptionalString(args, "state")
	secretType, _ := extractOptionalString(args, "secret_type")

	return &listSecretScanningAlertsRequest{
		Owner:      owner,
		Repo:       repo,
		State:      state,
		SecretType: secretType,
	}, nil
}

func handleListSecretScanningAlerts(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	req, err := parseListSecretScanningAlertsRequest(args)
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
	if req.SecretType != "" {
		params.Set("secret_type", req.SecretType)
	}

	var path, scope string
	if req.Repo != "" {
		path = fmt.Sprintf("/repos/%s/%s/secret-scanning/alerts", req.Owner, req.Repo)
		scope = req.Owner + "/" + req.Repo
	} else {
		path = fmt.Sprintf("/orgs/%s/secret-scanning/alerts", req.Owner)
		scope = req.Owner + " (org)"
	}

	rawItems, err := client.GetAllPages(ctx, path, params)
	if err != nil {
		msg := fmt.Sprintf("GitHub API error: %v", err)
		if req.Repo == "" && strings.Contains(err.Error(), "404") {
			msg += " — org-level secret scanning alerts require a GitHub organisation. For personal accounts, supply a 'repo' parameter."
		}
		return errorResult(msg), nil
	}

	alerts := make([]api.SecretScanningAlert, 0, len(rawItems))
	for _, raw := range rawItems {
		var a api.SecretScanningAlert
		if err := json.Unmarshal(raw, &a); err == nil {
			alerts = append(alerts, a)
		}
	}

	return buildSecretScanningListResult(scope, req, alerts), nil
}

func buildSecretScanningListResult(scope string, req *listSecretScanningAlertsRequest, alerts []api.SecretScanningAlert) interface{} {
	type alertSummary struct {
		Number                 int    `json:"number"`
		State                  string `json:"state"`
		SecretType             string `json:"secret_type"`
		SecretTypeDisplayName  string `json:"secret_type_display_name,omitempty"`
		Validity               string `json:"validity,omitempty"`
		Resolution             string `json:"resolution,omitempty"`
		PubliclyLeaked         bool   `json:"publicly_leaked"`
		MultiRepo              bool   `json:"multi_repo"`
		PushProtectionBypassed bool   `json:"push_protection_bypassed"`
		CreatedAt              string `json:"created_at"`
		HTMLURL                string `json:"html_url"`
	}

	summaries := make([]alertSummary, 0, len(alerts))
	for _, a := range alerts {
		summaries = append(summaries, alertSummary{
			Number:                 a.Number,
			State:                  a.State,
			SecretType:             a.SecretType,
			SecretTypeDisplayName:  a.SecretTypeDisplayName,
			Validity:               a.Validity,
			Resolution:             a.Resolution,
			PubliclyLeaked:         a.PubliclyLeaked,
			MultiRepo:              a.MultiRepo,
			PushProtectionBypassed: a.PushProtectionBypassed,
			CreatedAt:              a.CreatedAt,
			HTMLURL:                a.HTMLURL,
		})
	}

	filters := map[string]string{}
	if req.State != "" {
		filters["state"] = req.State
	}
	if req.SecretType != "" {
		filters["secret_type"] = req.SecretType
	}

	return map[string]interface{}{
		"scope":           scope,
		"total_count":     len(summaries),
		"filters_applied": filters,
		"alerts":          summaries,
	}
}
