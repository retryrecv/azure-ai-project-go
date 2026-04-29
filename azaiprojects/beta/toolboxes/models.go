package toolboxes

import (
	"encoding/json"

	"github.com/sambo/ai-projects-go/azaiprojects/internal/shared"
)

// ToolboxObject is a toolbox resource (without versions).
type ToolboxObject struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	DefaultVersion string `json:"default_version"`
}

// ToolboxVersionObject is a specific version of a toolbox.
//
// Tools and Policies are kept as json.RawMessage; ToolUnion and the nested
// RaiConfig union shape are pass-through — caller decodes when needed.
type ToolboxVersionObject struct {
	ID          string             `json:"id"`
	Name        string             `json:"name"`
	Version     string             `json:"version"`
	Description string             `json:"description,omitempty"`
	Metadata    map[string]string  `json:"metadata,omitempty"`
	CreatedAt   shared.UnixSeconds `json:"created_at"`
	Tools       []json.RawMessage  `json:"tools"`
	Policies    json.RawMessage    `json:"policies,omitempty"`
}

// CreateVersionBody is the JSON body for POST /toolboxes/{name}/versions.
type CreateVersionBody struct {
	Description string            `json:"description,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	Tools       []json.RawMessage `json:"tools"`
	Policies    json.RawMessage   `json:"policies,omitempty"`
}

// UpdateBody is the JSON body for PATCH /toolboxes/{name}.
type UpdateBody struct {
	DefaultVersion string `json:"default_version"`
}

// ToolboxesPage is one cursor page of toolboxes.
type ToolboxesPage struct {
	Data    []ToolboxObject `json:"data"`
	FirstID string          `json:"first_id,omitempty"`
	LastID  string          `json:"last_id,omitempty"`
	HasMore bool            `json:"has_more"`
}

// ToolboxVersionsPage is one cursor page of toolbox versions.
type ToolboxVersionsPage struct {
	Data    []ToolboxVersionObject `json:"data"`
	FirstID string                 `json:"first_id,omitempty"`
	LastID  string                 `json:"last_id,omitempty"`
	HasMore bool                   `json:"has_more"`
}

// PageOrder is the order parameter for cursor-paged list operations.
type PageOrder string

const (
	PageOrderAsc  PageOrder = "asc"
	PageOrderDesc PageOrder = "desc"
)
