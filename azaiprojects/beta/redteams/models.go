package redteams

import "encoding/json"

// RedTeam represents a red team run.
//
// AttackStrategies, RiskCategories, and Target are kept as json.RawMessage
// pass-throughs; the underlying TS unions (AttackStrategy nested arrays,
// RiskCategory enum, TargetConfigUnion with discriminator) are not
// flattened in this Go port.
//
// Note: the wire field is "id"; it is exposed as Name to mirror the TS surface.
type RedTeam struct {
	Name                string            `json:"id,omitempty"`
	DisplayName         string            `json:"displayName,omitempty"`
	NumTurns            *int32            `json:"numTurns,omitempty"`
	AttackStrategies    json.RawMessage   `json:"attackStrategies,omitempty"`
	SimulationOnly      *bool             `json:"simulationOnly,omitempty"`
	RiskCategories      json.RawMessage   `json:"riskCategories,omitempty"`
	ApplicationScenario string            `json:"applicationScenario,omitempty"`
	Tags                map[string]string `json:"tags,omitempty"`
	Properties          map[string]string `json:"properties,omitempty"`
	Status              string            `json:"status,omitempty"`
	Target              json.RawMessage   `json:"target,omitempty"`
}
