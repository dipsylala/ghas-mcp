package mcp_tools

import "fmt"

// extractRequiredString returns a non-empty string from args or an error.
func extractRequiredString(args map[string]interface{}, field string) (string, error) {
	val, ok := args[field].(string)
	if !ok || val == "" {
		return "", fmt.Errorf("%s is required", field)
	}
	return val, nil
}

// extractOptionalString returns an optional string from args.
// The second return value is false when the key is absent.
func extractOptionalString(args map[string]interface{}, field string) (string, bool) {
	val, ok := args[field].(string)
	return val, ok && val != ""
}

// extractInt extracts an integer from JSON-decoded args (float64 → int).
func extractInt(args map[string]interface{}, field string, defaultValue int) int {
	if val, ok := args[field].(float64); ok {
		return int(val)
	}
	return defaultValue
}
