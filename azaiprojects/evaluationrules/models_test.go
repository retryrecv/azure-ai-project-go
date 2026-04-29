package evaluationrules

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestEvaluationRuleFilter_RoundTrip(t *testing.T) {
	in := EvaluationRuleFilter{AgentName: "my-agent"}
	b, _ := json.Marshal(in)
	if string(b) != `{"agentName":"my-agent"}` {
		t.Fatalf("got %s", b)
	}
	var out EvaluationRuleFilter
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out != in {
		t.Fatalf("mismatch: %+v", out)
	}
}

func TestContinuousEvaluationRuleAction_RoundTrip(t *testing.T) {
	max := int32(10)
	in := ContinuousEvaluationRuleAction{
		Type:          EvaluationRuleActionTypeContinuousEvaluation,
		EvalID:        "eval_1",
		MaxHourlyRuns: &max,
	}
	b, _ := json.Marshal(in)
	if !strings.Contains(string(b), `"type":"continuousEvaluation"`) ||
		!strings.Contains(string(b), `"evalId":"eval_1"`) ||
		!strings.Contains(string(b), `"maxHourlyRuns":10`) {
		t.Fatalf("unexpected JSON: %s", b)
	}
	var out ContinuousEvaluationRuleAction
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.Type != in.Type || out.EvalID != in.EvalID || out.MaxHourlyRuns == nil || *out.MaxHourlyRuns != max {
		t.Fatalf("mismatch: %+v", out)
	}
}

func TestHumanEvaluationPreviewRuleAction_RoundTrip(t *testing.T) {
	in := HumanEvaluationPreviewRuleAction{
		Type:       EvaluationRuleActionTypeHumanEvaluationPreview,
		TemplateID: "tmpl_42",
	}
	b, _ := json.Marshal(in)
	want := `{"type":"humanEvaluationPreview","templateId":"tmpl_42"}`
	if string(b) != want {
		t.Fatalf("got %s want %s", b, want)
	}
	var out HumanEvaluationPreviewRuleAction
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out != in {
		t.Fatalf("mismatch: %+v", out)
	}
}

func TestEvaluationRuleActionValue_DispatchesByType(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		typ  EvaluationRuleActionType
		want any
	}{
		{
			"continuous",
			`{"type":"continuousEvaluation","evalId":"eval_1","maxHourlyRuns":3}`,
			EvaluationRuleActionTypeContinuousEvaluation,
			ContinuousEvaluationRuleAction{
				Type:          EvaluationRuleActionTypeContinuousEvaluation,
				EvalID:        "eval_1",
				MaxHourlyRuns: ptrInt32(3),
			},
		},
		{
			"human",
			`{"type":"humanEvaluationPreview","templateId":"t1"}`,
			EvaluationRuleActionTypeHumanEvaluationPreview,
			HumanEvaluationPreviewRuleAction{
				Type:       EvaluationRuleActionTypeHumanEvaluationPreview,
				TemplateID: "t1",
			},
		},
		{
			"base-fallback",
			`{"type":"unknown-future"}`,
			EvaluationRuleActionType("unknown-future"),
			EvaluationRuleAction{Type: "unknown-future"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var v EvaluationRuleActionValue
			if err := json.Unmarshal([]byte(tc.raw), &v); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if v.Type != tc.typ {
				t.Fatalf("type: got %s want %s", v.Type, tc.typ)
			}
			gotJSON, _ := json.Marshal(v.Value)
			wantJSON, _ := json.Marshal(tc.want)
			if string(gotJSON) != string(wantJSON) {
				t.Fatalf("typed value:\ngot  %s\nwant %s", gotJSON, wantJSON)
			}
			wrapped, err := json.Marshal(v)
			if err != nil {
				t.Fatalf("marshal wrapper: %v", err)
			}
			if string(wrapped) != string(gotJSON) {
				t.Fatalf("wrapper marshal:\ngot  %s\nwant %s", wrapped, gotJSON)
			}
		})
	}
}

func TestEvaluationRuleActionValue_NilMarshalsToNull(t *testing.T) {
	var v EvaluationRuleActionValue
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(b) != "null" {
		t.Fatalf("got %s want null", b)
	}
}

func TestEvaluationRule_RoundTrip(t *testing.T) {
	raw := []byte(`{
		"id":"rule_1",
		"displayName":"My Rule",
		"description":"demo",
		"action":{"type":"continuousEvaluation","evalId":"eval_42","maxHourlyRuns":5},
		"filter":{"agentName":"my-agent"},
		"eventType":"responseCompleted",
		"enabled":true,
		"systemData":{"createdAt":"2026-04-28T00:00:00Z"}
	}`)
	var r EvaluationRule
	if err := json.Unmarshal(raw, &r); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if r.ID != "rule_1" || r.DisplayName != "My Rule" || r.Description != "demo" ||
		r.Filter == nil || r.Filter.AgentName != "my-agent" ||
		r.EventType != EvaluationRuleEventTypeResponseCompleted || !r.Enabled ||
		r.SystemData["createdAt"] == "" {
		t.Fatalf("decoded fields: %+v", r)
	}
	if r.Action.Type != EvaluationRuleActionTypeContinuousEvaluation {
		t.Fatalf("action type: %s", r.Action.Type)
	}
	c, ok := r.Action.Value.(ContinuousEvaluationRuleAction)
	if !ok {
		t.Fatalf("action value: %T", r.Action.Value)
	}
	if c.EvalID != "eval_42" || c.MaxHourlyRuns == nil || *c.MaxHourlyRuns != 5 {
		t.Fatalf("action contents: %+v", c)
	}
	// Round-trip back to JSON and reparse.
	b, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var r2 EvaluationRule
	if err := json.Unmarshal(b, &r2); err != nil {
		t.Fatalf("re-unmarshal: %v", err)
	}
	if r2.ID != r.ID || r2.Action.Type != r.Action.Type {
		t.Fatalf("round-trip lost fields: %+v vs %+v", r2, r)
	}
}

func TestEvaluationRuleActionUnion_TypeMethods(t *testing.T) {
	cases := []struct {
		v    EvaluationRuleActionUnion
		want EvaluationRuleActionType
	}{
		{ContinuousEvaluationRuleAction{}, EvaluationRuleActionTypeContinuousEvaluation},
		{HumanEvaluationPreviewRuleAction{}, EvaluationRuleActionTypeHumanEvaluationPreview},
		{EvaluationRuleAction{}, EvaluationRuleActionType("")},
	}
	for _, tc := range cases {
		if got := tc.v.actionType(); got != tc.want {
			t.Fatalf("%T.actionType(): got %q want %q", tc.v, got, tc.want)
		}
	}
}

func ptrInt32(v int32) *int32 { return &v }
