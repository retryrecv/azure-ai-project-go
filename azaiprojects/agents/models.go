package agents

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

// AgentKind discriminates AgentDefinitionUnion variants.
type AgentKind string

const (
	AgentKindPrompt   AgentKind = "prompt"
	AgentKindHosted   AgentKind = "hosted"
	AgentKindWorkflow AgentKind = "workflow"
)

// AgentVersionStatus enumerates the lifecycle states of an AgentVersion.
type AgentVersionStatus string

const (
	AgentVersionStatusCreating AgentVersionStatus = "creating"
	AgentVersionStatusActive   AgentVersionStatus = "active"
	AgentVersionStatusFailed   AgentVersionStatus = "failed"
	AgentVersionStatusDeleting AgentVersionStatus = "deleting"
	AgentVersionStatusDeleted  AgentVersionStatus = "deleted"
)

// PageOrder sorts list responses by created_at.
type PageOrder string

const (
	PageOrderAsc  PageOrder = "asc"
	PageOrderDesc PageOrder = "desc"
)

// RaiConfig configures Responsible AI content filtering.
type RaiConfig struct {
	RaiPolicyName string `json:"rai_policy_name"`
}

// AgentIdentity is the principal/client identity assigned to an agent instance.
type AgentIdentity struct {
	PrincipalID string `json:"principal_id"`
	ClientID    string `json:"client_id"`
}

// AgentDefinition is the base shape of an agent definition. The Kind field
// discriminates the union (see AgentDefinitionValue for full union handling).
type AgentDefinition struct {
	Kind      AgentKind  `json:"kind"`
	RaiConfig *RaiConfig `json:"rai_config,omitempty"`
}

// UnixSeconds is a time.Time that round-trips JSON as Unix seconds (the wire
// format used by the agents API for created_at).
type UnixSeconds struct {
	time.Time
}

// MarshalJSON renders the time as integer Unix seconds.
func (t UnixSeconds) MarshalJSON() ([]byte, error) {
	if t.Time.IsZero() {
		return []byte("null"), nil
	}
	return []byte(strconv.FormatInt(t.Time.Unix(), 10)), nil
}

// UnmarshalJSON accepts an integer or float number of Unix seconds, or null.
func (t *UnixSeconds) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		t.Time = time.Time{}
		return nil
	}
	f, err := strconv.ParseFloat(string(data), 64)
	if err != nil {
		return fmt.Errorf("agents: UnixSeconds: %w", err)
	}
	sec := int64(f)
	nsec := int64((f - float64(sec)) * 1e9)
	t.Time = time.Unix(sec, nsec).UTC()
	return nil
}

// AgentVersion is one immutable version of an agent definition.
type AgentVersion struct {
	Object             string             `json:"object"` // always "agent.version"
	ID                 string             `json:"id"`
	Name               string             `json:"name"`
	Version            string             `json:"version"`
	Description        string             `json:"description,omitempty"`
	Metadata           map[string]string  `json:"metadata,omitempty"`
	CreatedAt          UnixSeconds        `json:"created_at"`
	Definition         json.RawMessage    `json:"definition"` // typed via AgentDefinitionValue once #12 lands
	Status             AgentVersionStatus `json:"status,omitempty"`
	InstanceIdentity   *AgentIdentity     `json:"instance_identity,omitempty"`
	Blueprint          *AgentIdentity     `json:"blueprint,omitempty"`
	BlueprintReference json.RawMessage    `json:"blueprint_reference,omitempty"` // typed in #20
	AgentGUID          string             `json:"agent_guid,omitempty"`
}

// AgentVersionsRef is the {latest: AgentVersion} wrapper used inside Agent.
type AgentVersionsRef struct {
	Latest AgentVersion `json:"latest"`
}

// Agent is the top-level agent record returned by the service.
type Agent struct {
	Object             string            `json:"object"` // always "agent"
	ID                 string            `json:"id"`
	Name               string            `json:"name"`
	Versions           AgentVersionsRef  `json:"versions"`
	AgentEndpoint      json.RawMessage   `json:"agent_endpoint,omitempty"`      // typed in #20
	InstanceIdentity   *AgentIdentity    `json:"instance_identity,omitempty"`
	Blueprint          *AgentIdentity    `json:"blueprint,omitempty"`
	BlueprintReference json.RawMessage   `json:"blueprint_reference,omitempty"` // typed in #20
	AgentCard          json.RawMessage   `json:"agent_card,omitempty"`          // typed in #20
}
