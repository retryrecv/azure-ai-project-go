package evaluationrules

import (
	"encoding/json"
	"fmt"
)

// EvaluationRuleEventType enumerates the events an evaluation rule can react to.
type EvaluationRuleEventType string

const (
	EvaluationRuleEventTypeResponseCompleted EvaluationRuleEventType = "responseCompleted"
	EvaluationRuleEventTypeManual            EvaluationRuleEventType = "manual"
)

// EvaluationRuleActionType discriminates EvaluationRuleActionUnion variants.
type EvaluationRuleActionType string

const (
	EvaluationRuleActionTypeContinuousEvaluation     EvaluationRuleActionType = "continuousEvaluation"
	EvaluationRuleActionTypeHumanEvaluationPreview   EvaluationRuleActionType = "humanEvaluationPreview"
)

// EvaluationRuleFilter narrows which traffic an evaluation rule applies to.
type EvaluationRuleFilter struct {
	AgentName string `json:"agentName"`
}

// EvaluationRuleAction is the base shape of an evaluation-rule action. The Type
// field discriminates the union (see EvaluationRuleActionValue for typed dispatch).
type EvaluationRuleAction struct {
	Type EvaluationRuleActionType `json:"type"`
}

// ContinuousEvaluationRuleAction queues continuous-evaluation runs against an eval.
type ContinuousEvaluationRuleAction struct {
	Type           EvaluationRuleActionType `json:"type"` // always continuousEvaluation
	EvalID         string                   `json:"evalId"`
	MaxHourlyRuns  *int32                   `json:"maxHourlyRuns,omitempty"`
}

// HumanEvaluationPreviewRuleAction routes traffic into a human-evaluation template.
type HumanEvaluationPreviewRuleAction struct {
	Type       EvaluationRuleActionType `json:"type"` // always humanEvaluationPreview
	TemplateID string                   `json:"templateId"`
}

// EvaluationRuleActionUnion is implemented by every concrete action variant.
// MarshalJSON on each variant writes the right shape; the package helpers
// (UnmarshalJSON on EvaluationRuleActionValue) handle the inverse.
type EvaluationRuleActionUnion interface {
	actionType() EvaluationRuleActionType
}

func (EvaluationRuleAction) actionType() EvaluationRuleActionType { return "" }
func (ContinuousEvaluationRuleAction) actionType() EvaluationRuleActionType {
	return EvaluationRuleActionTypeContinuousEvaluation
}
func (HumanEvaluationPreviewRuleAction) actionType() EvaluationRuleActionType {
	return EvaluationRuleActionTypeHumanEvaluationPreview
}

// EvaluationRuleActionValue holds a decoded EvaluationRuleActionUnion plus the
// original JSON. Use it as the destination for fields like EvaluationRule.Action
// when you want a typed view of the union.
type EvaluationRuleActionValue struct {
	Type  EvaluationRuleActionType
	Value EvaluationRuleActionUnion
	Raw   json.RawMessage
}

// UnmarshalJSON dispatches on the "type" field.
func (v *EvaluationRuleActionValue) UnmarshalJSON(data []byte) error {
	v.Raw = append(v.Raw[:0], data...)
	var probe struct {
		Type EvaluationRuleActionType `json:"type"`
	}
	if err := json.Unmarshal(data, &probe); err != nil {
		return fmt.Errorf("evaluationrules: probe action type: %w", err)
	}
	v.Type = probe.Type
	switch probe.Type {
	case EvaluationRuleActionTypeContinuousEvaluation:
		var c ContinuousEvaluationRuleAction
		if err := json.Unmarshal(data, &c); err != nil {
			return fmt.Errorf("evaluationrules: ContinuousEvaluationRuleAction: %w", err)
		}
		v.Value = c
	case EvaluationRuleActionTypeHumanEvaluationPreview:
		var h HumanEvaluationPreviewRuleAction
		if err := json.Unmarshal(data, &h); err != nil {
			return fmt.Errorf("evaluationrules: HumanEvaluationPreviewRuleAction: %w", err)
		}
		v.Value = h
	default:
		var b EvaluationRuleAction
		if err := json.Unmarshal(data, &b); err != nil {
			return fmt.Errorf("evaluationrules: EvaluationRuleAction (base): %w", err)
		}
		v.Value = b
	}
	return nil
}

// MarshalJSON delegates to the underlying typed value.
func (v EvaluationRuleActionValue) MarshalJSON() ([]byte, error) {
	if v.Value == nil {
		return []byte("null"), nil
	}
	return json.Marshal(v.Value)
}

// EvaluationRule is one configured evaluation rule.
type EvaluationRule struct {
	ID          string                    `json:"id,omitempty"` // service-assigned; omit on create
	DisplayName string                    `json:"displayName,omitempty"`
	Description string                    `json:"description,omitempty"`
	Action      EvaluationRuleActionValue `json:"action"`
	Filter      *EvaluationRuleFilter     `json:"filter,omitempty"`
	EventType   EvaluationRuleEventType   `json:"eventType"`
	Enabled     bool                      `json:"enabled"`
	SystemData  map[string]string         `json:"systemData,omitempty"`
}
