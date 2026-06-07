// Package api contains GitHub REST API response types for GHAS alert endpoints.
package api

import "encoding/json"

// ---- Shared types ----

// User represents a minimal GitHub user object.
type User struct {
	Login string `json:"login"`
}

// ---- Code scanning ----

// CodeScanningAlert represents a single code scanning alert from GitHub.
type CodeScanningAlert struct {
	Number    int    `json:"number"`
	State     string `json:"state"` // open | dismissed | fixed
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
	FixedAt   string `json:"fixed_at,omitempty"`

	DismissedAt     string `json:"dismissed_at,omitempty"`
	DismissedReason string `json:"dismissed_reason,omitempty"`
	DismissedBy     *User  `json:"dismissed_by,omitempty"`

	Rule               CodeScanningRule      `json:"rule"`
	Tool               CodeScanningTool      `json:"tool"`
	HTMLURL            string                `json:"html_url"`
	MostRecentInstance *CodeScanningInstance `json:"most_recent_instance,omitempty"`
}

// CodeScanningRule describes the rule that triggered a code scanning alert.
type CodeScanningRule struct {
	ID                    string   `json:"id"`
	Name                  string   `json:"name"`
	Severity              string   `json:"severity"`                          // none | note | warning | error
	SecuritySeverityLevel string   `json:"security_severity_level,omitempty"` // low | medium | high | critical
	Description           string   `json:"description"`
	FullDescription       string   `json:"full_description,omitempty"`
	Tags                  []string `json:"tags,omitempty"`
	Help                  string   `json:"help,omitempty"`
}

// CodeScanningTool describes the analysis tool that produced a code scanning alert.
type CodeScanningTool struct {
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
}

// CodeScanningInstance represents a single instance (occurrence) of a code scanning alert.
type CodeScanningInstance struct {
	Ref             string                `json:"ref"`
	State           string                `json:"state"`
	CommitSHA       string                `json:"commit_sha,omitempty"`
	Message         *CodeScanningMessage  `json:"message,omitempty"`
	Location        *CodeScanningLocation `json:"location,omitempty"`
	Classifications []string              `json:"classifications,omitempty"`
}

// CodeScanningMessage is the human-readable message for an alert instance.
type CodeScanningMessage struct {
	Text string `json:"text"`
}

// CodeScanningLocation pinpoints where in the code an alert was found.
type CodeScanningLocation struct {
	Path        string `json:"path"`
	StartLine   int    `json:"start_line"`
	EndLine     int    `json:"end_line"`
	StartColumn int    `json:"start_column,omitempty"`
	EndColumn   int    `json:"end_column,omitempty"`
}

// ---- Dependabot ----

// DependabotAlert represents a single Dependabot vulnerability alert.
type DependabotAlert struct {
	Number    int    `json:"number"`
	State     string `json:"state"`    // open | dismissed | fixed | auto_dismissed
	Severity  string `json:"severity"` // low | medium | high | critical
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
	FixedAt   string `json:"fixed_at,omitempty"`

	DismissedAt      string `json:"dismissed_at,omitempty"`
	DismissedReason  string `json:"dismissed_reason,omitempty"`
	DismissedComment string `json:"dismissed_comment,omitempty"`
	DismissedBy      *User  `json:"dismissed_by,omitempty"`

	Dependency            DependabotDependency       `json:"dependency"`
	SecurityAdvisory      DependabotSecurityAdvisory `json:"security_advisory"`
	SecurityVulnerability DependabotVulnerability    `json:"security_vulnerability"`
	HTMLURL               string                     `json:"html_url"`
}

// DependabotDependency identifies the vulnerable package.
type DependabotDependency struct {
	Package      DependabotPackage `json:"package"`
	ManifestPath string            `json:"manifest_path,omitempty"`
	Scope        string            `json:"scope,omitempty"` // runtime | development
}

// DependabotPackage is an ecosystem + package name pair.
type DependabotPackage struct {
	Ecosystem string `json:"ecosystem"`
	Name      string `json:"name"`
}

// DependabotSecurityAdvisory contains NVD/GHSA advisory details.
type DependabotSecurityAdvisory struct {
	GHSAID      string          `json:"ghsa_id"`
	CVEID       string          `json:"cve_id,omitempty"`
	Summary     string          `json:"summary"`
	Description string          `json:"description"`
	Severity    string          `json:"severity"`
	CVSS        *DependabotCVSS `json:"cvss,omitempty"`
	EPSS        *DependabotEPSS `json:"epss,omitempty"`
	CWEs        []DependabotCWE `json:"cwes,omitempty"`
	PublishedAt string          `json:"published_at,omitempty"`
	UpdatedAt   string          `json:"updated_at,omitempty"`
}

// DependabotCVSS holds CVSS scoring information.
type DependabotCVSS struct {
	Score        float64 `json:"score"`
	VectorString string  `json:"vector_string,omitempty"`
}

// DependabotEPSS holds EPSS probability data.
type DependabotEPSS struct {
	Percentage float64 `json:"percentage"`
	Percentile float64 `json:"percentile"`
}

// DependabotCWE identifies a weakness associated with an advisory.
type DependabotCWE struct {
	CWEID string `json:"cwe_id"`
	Name  string `json:"name"`
}

// DependabotVulnerability describes the specific vulnerable version range.
type DependabotVulnerability struct {
	Package                DependabotPackage         `json:"package"`
	Severity               string                    `json:"severity"`
	VulnerableVersionRange string                    `json:"vulnerable_version_range,omitempty"`
	FirstPatchedVersion    *DependabotPatchedVersion `json:"first_patched_version,omitempty"`
}

// DependabotPatchedVersion holds the first version that fixes a vulnerability.
type DependabotPatchedVersion struct {
	Identifier string `json:"identifier"`
}

// ---- Secret scanning ----

// SecretScanningAlert represents a single secret scanning alert.
type SecretScanningAlert struct {
	Number    int    `json:"number"`
	State     string `json:"state"` // open | resolved
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`

	Resolution        string `json:"resolution,omitempty"`
	ResolvedAt        string `json:"resolved_at,omitempty"`
	ResolvedBy        *User  `json:"resolved_by,omitempty"`
	ResolutionComment string `json:"resolution_comment,omitempty"`

	SecretType            string `json:"secret_type"`
	SecretTypeDisplayName string `json:"secret_type_display_name,omitempty"`
	// Secret is intentionally excluded to avoid leaking values through LLM context.

	Validity               string `json:"validity,omitempty"` // active | inactive | unknown
	PushProtectionBypassed bool   `json:"push_protection_bypassed"`
	PubliclyLeaked         bool   `json:"publicly_leaked"`
	MultiRepo              bool   `json:"multi_repo"`
	HTMLURL                string `json:"html_url"`
}

// ---- Pagination ----

// PagedResponse is used internally to decode list responses with Link headers.
// The GitHub REST API uses RFC 5988 Link headers for pagination (not a body envelope).
type PagedResponse[T any] struct {
	Items   []T
	HasNext bool
}

// UnmarshalJSONArray is a helper to decode a top-level JSON array into a slice.
func UnmarshalJSONArray[T any](data []byte) ([]T, error) {
	var items []T
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, err
	}
	return items, nil
}
