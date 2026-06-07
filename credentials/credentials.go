// Package credentials provides GitHub PAT loading from environment variables
// or the gh CLI.
package credentials

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

const (
	// DefaultBaseURL is the GitHub REST API base URL.
	DefaultBaseURL = "https://api.github.com"
)

// GetToken returns a GitHub PAT.
// Resolution order:
//  1. GITHUB_TOKEN environment variable
//  2. `gh auth token` (works on all platforms, including Windows where the
//     token is stored in the system credential manager rather than a plain file)
func GetToken() (string, error) {
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		return token, nil
	}

	token, err := ghCLIToken()
	if err == nil && token != "" {
		return token, nil
	}

	return "", fmt.Errorf(
		"GitHub token not found: set GITHUB_TOKEN environment variable " +
			"or authenticate with the gh CLI (gh auth login)",
	)
}

// ghCLIToken retrieves the active token by running `gh auth token`.
// This works on all platforms regardless of where gh stores credentials.
func ghCLIToken() (string, error) {
	out, err := exec.Command("gh", "auth", "token").Output() // nolint:gosec // Fixed args, no user input
	if err != nil {
		return "", fmt.Errorf("gh auth token: %w", err)
	}
	token := strings.TrimSpace(string(out))
	if token == "" {
		return "", fmt.Errorf("gh auth token returned empty output")
	}
	return token, nil
}
