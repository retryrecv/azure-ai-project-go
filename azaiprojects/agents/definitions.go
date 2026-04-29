package agents

import (
	"encoding/json"
	"fmt"
)

// Reasoning configures gpt-5/o-series reasoning models on a prompt agent.
type Reasoning struct {
	Effort          ReasoningEffort `json:"effort,omitempty"`
	Summary         string          `json:"summary,omitempty"`          // auto|concise|detailed
	GenerateSummary string          `json:"generate_summary,omitempty"` // auto|concise|detailed
}

// ReasoningEffort controls how much reasoning effort the model spends.
type ReasoningEffort string

const (
	ReasoningEffortNone    ReasoningEffort = "none"
	ReasoningEffortMinimal ReasoningEffort = "minimal"
	ReasoningEffortLow     ReasoningEffort = "low"
	ReasoningEffortMedium  ReasoningEffort = "medium"
	ReasoningEffortHigh    ReasoningEffort = "high"
	ReasoningEffortXHigh   ReasoningEffort = "xhigh"
)

// PromptAgentDefinition is an agent backed by a chat-completions style model.
//
// Tools/ToolChoice/Text/StructuredInputs are kept as json.RawMessage so callers
// can round-trip arbitrary tool unions without this package having to model the
// 30+ tool variants. Encode/decode them yourself when the time comes.
type PromptAgentDefinition struct {
	Kind             AgentKind         `json:"kind"` // always AgentKindPrompt
	RaiConfig        *RaiConfig        `json:"rai_config,omitempty"`
	Model            string            `json:"model"`
	Instructions     string            `json:"instructions,omitempty"`
	Temperature      *float64          `json:"temperature,omitempty"`
	TopP             *float64          `json:"top_p,omitempty"`
	Reasoning        *Reasoning        `json:"reasoning,omitempty"`
	Tools            json.RawMessage   `json:"tools,omitempty"`
	ToolChoice       json.RawMessage   `json:"tool_choice,omitempty"`
	Text             json.RawMessage   `json:"text,omitempty"`
	StructuredInputs json.RawMessage   `json:"structured_inputs,omitempty"`
}

// HostedAgentDefinition is an agent that runs as a hosted container or code app.
//
// ContainerConfiguration/CodeConfiguration/ProtocolVersions are pass-through
// json.RawMessage; see the package docs for upgrading specific fields.
type HostedAgentDefinition struct {
	Kind                      AgentKind         `json:"kind"` // always AgentKindHosted
	RaiConfig                 *RaiConfig        `json:"rai_config,omitempty"`
	Tools                     json.RawMessage   `json:"tools,omitempty"`
	ContainerProtocolVersions json.RawMessage   `json:"container_protocol_versions,omitempty"`
	CPU                       string            `json:"cpu"`
	Memory                    string            `json:"memory"`
	EnvironmentVariables      map[string]string `json:"environment_variables,omitempty"`
	Image                     string            `json:"image,omitempty"`
	ContainerConfiguration    json.RawMessage   `json:"container_configuration,omitempty"`
	ProtocolVersions          json.RawMessage   `json:"protocol_versions,omitempty"`
	CodeConfiguration         json.RawMessage   `json:"code_configuration,omitempty"`
}

// WorkflowAgentDefinition is an agent defined by a CSDL YAML workflow.
type WorkflowAgentDefinition struct {
	Kind      AgentKind  `json:"kind"` // always AgentKindWorkflow
	RaiConfig *RaiConfig `json:"rai_config,omitempty"`
	Workflow  string     `json:"workflow,omitempty"`
}

// AgentDefinitionUnion is implemented by every concrete agent definition.
// MarshalAgentDefinition writes the value's kind discriminator; the package
// helpers (UnmarshalAgentDefinition / AgentDefinitionValue) handle the inverse.
type AgentDefinitionUnion interface {
	agentKind() AgentKind
}

func (PromptAgentDefinition) agentKind() AgentKind   { return AgentKindPrompt }
func (HostedAgentDefinition) agentKind() AgentKind   { return AgentKindHosted }
func (WorkflowAgentDefinition) agentKind() AgentKind { return AgentKindWorkflow }
func (AgentDefinition) agentKind() AgentKind         { return "" } // base / fallback

// AgentDefinitionValue holds a decoded AgentDefinitionUnion plus the original
// JSON. Use it as the destination for fields like AgentVersion.Definition when
// you want a typed view of the union.
type AgentDefinitionValue struct {
	Kind  AgentKind
	Value AgentDefinitionUnion
	Raw   json.RawMessage
}

// UnmarshalJSON dispatches on the "kind" field.
func (v *AgentDefinitionValue) UnmarshalJSON(data []byte) error {
	v.Raw = append(v.Raw[:0], data...)
	var probe struct {
		Kind AgentKind `json:"kind"`
	}
	if err := json.Unmarshal(data, &probe); err != nil {
		return fmt.Errorf("agents: AgentDefinitionValue: probe kind: %w", err)
	}
	v.Kind = probe.Kind
	switch probe.Kind {
	case AgentKindPrompt:
		var p PromptAgentDefinition
		if err := json.Unmarshal(data, &p); err != nil {
			return fmt.Errorf("agents: PromptAgentDefinition: %w", err)
		}
		v.Value = p
	case AgentKindHosted:
		var h HostedAgentDefinition
		if err := json.Unmarshal(data, &h); err != nil {
			return fmt.Errorf("agents: HostedAgentDefinition: %w", err)
		}
		v.Value = h
	case AgentKindWorkflow:
		var w WorkflowAgentDefinition
		if err := json.Unmarshal(data, &w); err != nil {
			return fmt.Errorf("agents: WorkflowAgentDefinition: %w", err)
		}
		v.Value = w
	default:
		var b AgentDefinition
		if err := json.Unmarshal(data, &b); err != nil {
			return fmt.Errorf("agents: AgentDefinition (base): %w", err)
		}
		v.Value = b
	}
	return nil
}

// MarshalJSON delegates to the underlying typed value.
func (v AgentDefinitionValue) MarshalJSON() ([]byte, error) {
	if v.Value == nil {
		return []byte("null"), nil
	}
	return json.Marshal(v.Value)
}

// DecodeDefinition decodes the raw JSON of an AgentVersion.Definition into a
// typed AgentDefinitionValue. Returns an empty value if the source is empty.
func DecodeDefinition(raw json.RawMessage) (AgentDefinitionValue, error) {
	if len(raw) == 0 {
		return AgentDefinitionValue{}, nil
	}
	var v AgentDefinitionValue
	if err := json.Unmarshal(raw, &v); err != nil {
		return AgentDefinitionValue{}, err
	}
	return v, nil
}
