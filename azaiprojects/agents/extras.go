package agents

import "encoding/json"

// AgentBlueprintReferenceType discriminates AgentBlueprintReference variants.
// Today only "ManagedAgentIdentityBlueprint" exists.
type AgentBlueprintReferenceType string

const (
	AgentBlueprintReferenceTypeManagedAgentIdentity AgentBlueprintReferenceType = "ManagedAgentIdentityBlueprint"
)

// AgentBlueprintReference references the blueprint an agent was instantiated
// from. Currently the only typed variant is ManagedAgentIdentityBlueprint, so
// BlueprintID is just a top-level field; new variants would warrant a wrapper.
type AgentBlueprintReference struct {
	Type        AgentBlueprintReferenceType `json:"type"`
	BlueprintID string                      `json:"blueprint_id,omitempty"`
}

// AgentEndpointProtocol enumerates supported endpoint protocols.
type AgentEndpointProtocol string

const (
	AgentEndpointProtocolActivity    AgentEndpointProtocol = "activity"
	AgentEndpointProtocolResponses   AgentEndpointProtocol = "responses"
	AgentEndpointProtocolA2A         AgentEndpointProtocol = "a2a"
	AgentEndpointProtocolInvocations AgentEndpointProtocol = "invocations"
)

// AgentEndpoint configures the routing/auth surface of an agent endpoint.
//
// VersionSelector and AuthorizationSchemes are kept as json.RawMessage because
// each is its own discriminated-union tree (FixedRatio rules; Entra/BotService/
// BotServiceRbac schemes with further nested unions). Decode the raw payload
// into your own types if you need them.
type AgentEndpoint struct {
	VersionSelector      json.RawMessage         `json:"version_selector,omitempty"`
	Protocols            []AgentEndpointProtocol `json:"protocols,omitempty"`
	AuthorizationSchemes []json.RawMessage       `json:"authorization_schemes,omitempty"`
}

// AgentCardSkill describes one skill the agent advertises.
type AgentCardSkill struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	Examples    []string `json:"examples,omitempty"`
}

// AgentCard is the public-facing capability description for an agent.
type AgentCard struct {
	Version     string           `json:"version"`
	Description string           `json:"description,omitempty"`
	Skills      []AgentCardSkill `json:"skills"`
}

// DeleteAgentResponse is returned by Delete.
type DeleteAgentResponse struct {
	Object  string `json:"object"` // always "agent.deleted"
	Name    string `json:"name"`
	Deleted bool   `json:"deleted"`
}

// DeleteAgentVersionResponse is returned by DeleteVersion.
type DeleteAgentVersionResponse struct {
	Object  string `json:"object"` // always "agent.version.deleted"
	Name    string `json:"name"`
	Version string `json:"version"`
	Deleted bool   `json:"deleted"`
}

// AgentsPage is one page of an agents list response. Pagination is cursor-based:
// when HasMore is true, pass LastID as the next request's "after" param.
type AgentsPage struct {
	Data    []Agent `json:"data"`
	FirstID string  `json:"first_id,omitempty"`
	LastID  string  `json:"last_id,omitempty"`
	HasMore bool    `json:"has_more"`
}

// AgentVersionsPage is one page of an agent-versions list response.
type AgentVersionsPage struct {
	Data    []AgentVersion `json:"data"`
	FirstID string         `json:"first_id,omitempty"`
	LastID  string         `json:"last_id,omitempty"`
	HasMore bool           `json:"has_more"`
}

// agentManifestBody is the on-the-wire shape for /agents:import.
type agentManifestBody struct {
	Name            string         `json:"name"`
	Metadata        map[string]string `json:"metadata,omitempty"`
	Description     string         `json:"description,omitempty"`
	ManifestID      string         `json:"manifest_id"`
	ParameterValues map[string]any `json:"parameter_values"`
}

// agentVersionManifestBody is the on-the-wire shape for the two manifest
// endpoints under /agents/{name}/{import,versions:import}.
type agentVersionManifestBody struct {
	Metadata        map[string]string `json:"metadata,omitempty"`
	Description     string         `json:"description,omitempty"`
	ManifestID      string         `json:"manifest_id"`
	ParameterValues map[string]any `json:"parameter_values"`
}
