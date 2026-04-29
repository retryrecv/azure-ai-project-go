package agents

import "encoding/json"

// Agent represents an agent endpoint resource.
//
// AgentEndpoint and AgentCard are kept as json.RawMessage; the underlying
// TS shapes (AgentEndpoint, AgentCard) are not flattened in this Go port.
type Agent struct {
	Name          string          `json:"name,omitempty"`
	AgentEndpoint json.RawMessage `json:"agent_endpoint,omitempty"`
	AgentCard     json.RawMessage `json:"agent_card,omitempty"`
}

// PatchAgentBody is the JSON body for PATCH /agents/{agent_name}
// (content-type: application/merge-patch+json).
type PatchAgentBody struct {
	AgentEndpoint json.RawMessage `json:"agent_endpoint,omitempty"`
	AgentCard     json.RawMessage `json:"agent_card,omitempty"`
}

// AgentSessionResource represents a session for an agent endpoint.
type AgentSessionResource struct {
	SessionID        string          `json:"session_id,omitempty"`
	AgentSessionID   string          `json:"agent_session_id,omitempty"`
	VersionIndicator json.RawMessage `json:"version_indicator,omitempty"`
}

// SessionsPage is one cursor page returned by GET .../sessions.
type SessionsPage struct {
	Data    []AgentSessionResource `json:"data"`
	FirstID string                 `json:"first_id,omitempty"`
	LastID  string                 `json:"last_id,omitempty"`
	HasMore bool                   `json:"has_more"`
}

// PageOrder is the order parameter for cursor-paged session list operations.
type PageOrder string

const (
	PageOrderAsc  PageOrder = "asc"
	PageOrderDesc PageOrder = "desc"
)

// CreateSessionBody is the JSON body for POST .../sessions.
type CreateSessionBody struct {
	AgentSessionID   string          `json:"agent_session_id,omitempty"`
	VersionIndicator json.RawMessage `json:"version_indicator,omitempty"`
}

// SessionDirectoryListResponse is the response of GET .../files (list).
//
// Entries is kept as json.RawMessage; the underlying TS type is an array
// of SessionDirectoryEntry objects which is not flattened here.
type SessionDirectoryListResponse struct {
	Path    string          `json:"path,omitempty"`
	Entries json.RawMessage `json:"entries,omitempty"`
}

// SessionFileWriteResponse is the response of PUT .../files/content.
type SessionFileWriteResponse struct {
	Path string `json:"path,omitempty"`
	Size int64  `json:"size,omitempty"`
}
