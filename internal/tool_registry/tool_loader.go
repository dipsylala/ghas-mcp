// Package tools provides tool definition loading and management.
package tools

import (
	"encoding/json"
	"fmt"

	"github.com/dipsylala/ghas-mcp/internal/types"
)

var toolsJSON []byte

// SetToolsJSON stores the embedded tools.json data.
func SetToolsJSON(data []byte) { toolsJSON = data }

// ToolDefinition represents one tool from tools.json.
type ToolDefinition struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Params      []ParamDefinition `json:"params"`
}

// ParamDefinition represents one parameter in a tool definition.
type ParamDefinition struct {
	Name          string           `json:"name"`
	Type          string           `json:"type"`
	IsRequired    bool             `json:"isRequired"`
	AllowedValues []string         `json:"allowedValues,omitempty"`
	Validation    *ValidationRules `json:"validation,omitempty"`
	Description   string           `json:"description"`
}

// ValidationRules holds min/max constraints.
type ValidationRules struct {
	Min *float64 `json:"min,omitempty"`
	Max *float64 `json:"max,omitempty"`
}

// ToolRegistry holds all tool definitions.
type ToolRegistry struct {
	Tools []ToolDefinition `json:"tools"`
}

// LoadToolDefinitions parses the embedded tools.json.
func LoadToolDefinitions() (*ToolRegistry, error) {
	var reg ToolRegistry
	if err := json.Unmarshal(toolsJSON, &reg); err != nil {
		return nil, fmt.Errorf("failed to parse tools.json: %w", err)
	}
	return &reg, nil
}

// GetToolByName looks up a tool by name; returns nil if not found.
func (r *ToolRegistry) GetToolByName(name string) *ToolDefinition {
	for i := range r.Tools {
		if r.Tools[i].Name == name {
			return &r.Tools[i]
		}
	}
	return nil
}

// GetAllMCPTools converts all definitions to MCP Tool format.
func (r *ToolRegistry) GetAllMCPTools() []types.Tool {
	out := make([]types.Tool, 0, len(r.Tools))
	for _, td := range r.Tools {
		out = append(out, td.ToMCPTool())
	}
	return out
}

// ToMCPTool converts a ToolDefinition to the MCP protocol Tool shape.
func (td *ToolDefinition) ToMCPTool() types.Tool {
	properties := make(map[string]interface{})
	required := []string{}

	for _, p := range td.Params {
		prop := map[string]interface{}{"description": p.Description}

		switch p.Type {
		case "string":
			prop["type"] = "string"
			if len(p.AllowedValues) > 0 {
				prop["enum"] = p.AllowedValues
			}
		case "number", "integer":
			prop["type"] = p.Type
			if p.Validation != nil {
				if p.Validation.Min != nil {
					prop["minimum"] = *p.Validation.Min
				}
				if p.Validation.Max != nil {
					prop["maximum"] = *p.Validation.Max
				}
			}
		default:
			prop["type"] = p.Type
		}

		properties[p.Name] = prop
		if p.IsRequired {
			required = append(required, p.Name)
		}
	}

	schema := map[string]interface{}{
		"type":       "object",
		"properties": properties,
	}
	if len(required) > 0 {
		schema["required"] = required
	}

	return types.Tool{
		Name:        td.Name,
		Description: td.Description,
		InputSchema: schema,
	}
}
