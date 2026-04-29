package agents

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestPromptAgentDefinition_RoundTrip(t *testing.T) {
	temp, topP := 0.5, 0.9
	in := PromptAgentDefinition{
		Kind:         AgentKindPrompt,
		Model:        "gpt-5.4",
		Instructions: "Be helpful.",
		Temperature:  &temp,
		TopP:         &topP,
		Reasoning:    &Reasoning{Effort: ReasoningEffortMedium, Summary: "concise"},
		RaiConfig:    &RaiConfig{RaiPolicyName: "default"},
		Tools:        json.RawMessage(`[{"type":"function","name":"foo"}]`),
	}
	b, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	s := string(b)
	for _, want := range []string{`"kind":"prompt"`, `"model":"gpt-5.4"`, `"temperature":0.5`,
		`"top_p":0.9`, `"reasoning":{`, `"rai_policy_name":"default"`, `"tools":[`} {
		if !strings.Contains(s, want) {
			t.Fatalf("missing %s in JSON: %s", want, s)
		}
	}
	var out PromptAgentDefinition
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.Kind != AgentKindPrompt || out.Model != in.Model ||
		*out.Temperature != temp || *out.TopP != topP ||
		out.Reasoning == nil || out.Reasoning.Effort != ReasoningEffortMedium {
		t.Fatalf("decoded mismatch: %+v", out)
	}
}

func TestHostedAgentDefinition_RoundTrip(t *testing.T) {
	in := HostedAgentDefinition{
		Kind:                 AgentKindHosted,
		CPU:                  "1",
		Memory:               "2Gi",
		Image:                "myreg/myimg:latest",
		EnvironmentVariables: map[string]string{"FOO": "bar"},
	}
	b, _ := json.Marshal(in)
	if !strings.Contains(string(b), `"kind":"hosted"`) ||
		!strings.Contains(string(b), `"cpu":"1"`) ||
		!strings.Contains(string(b), `"memory":"2Gi"`) ||
		!strings.Contains(string(b), `"image":"myreg/myimg:latest"`) ||
		!strings.Contains(string(b), `"environment_variables":{"FOO":"bar"}`) {
		t.Fatalf("unexpected JSON: %s", b)
	}
	var out HostedAgentDefinition
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.CPU != in.CPU || out.Memory != in.Memory || out.Image != in.Image ||
		out.EnvironmentVariables["FOO"] != "bar" {
		t.Fatalf("mismatch: %+v", out)
	}
}

func TestWorkflowAgentDefinition_RoundTrip(t *testing.T) {
	in := WorkflowAgentDefinition{
		Kind:     AgentKindWorkflow,
		Workflow: "name: hello\nsteps: []\n",
	}
	b, _ := json.Marshal(in)
	want := `{"kind":"workflow","workflow":"name: hello\nsteps: []\n"}`
	if string(b) != want {
		t.Fatalf("got %s want %s", b, want)
	}
	var out WorkflowAgentDefinition
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out != in {
		t.Fatalf("mismatch: %+v", out)
	}
}

func TestAgentDefinitionValue_DispatchesByKind(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		kind AgentKind
		want any
	}{
		{
			"prompt",
			`{"kind":"prompt","model":"m"}`,
			AgentKindPrompt,
			PromptAgentDefinition{Kind: AgentKindPrompt, Model: "m"},
		},
		{
			"hosted",
			`{"kind":"hosted","cpu":"1","memory":"2Gi"}`,
			AgentKindHosted,
			HostedAgentDefinition{Kind: AgentKindHosted, CPU: "1", Memory: "2Gi"},
		},
		{
			"workflow",
			`{"kind":"workflow","workflow":"x"}`,
			AgentKindWorkflow,
			WorkflowAgentDefinition{Kind: AgentKindWorkflow, Workflow: "x"},
		},
		{
			"base-fallback",
			`{"kind":"unknown-future","rai_config":{"rai_policy_name":"p"}}`,
			AgentKind("unknown-future"),
			AgentDefinition{Kind: "unknown-future", RaiConfig: &RaiConfig{RaiPolicyName: "p"}},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			v, err := DecodeDefinition(json.RawMessage(tc.raw))
			if err != nil {
				t.Fatalf("decode: %v", err)
			}
			if v.Kind != tc.kind {
				t.Fatalf("kind: got %s want %s", v.Kind, tc.kind)
			}
			// Compare via re-marshal (handles nested *RaiConfig pointer equality).
			gotJSON, _ := json.Marshal(v.Value)
			wantJSON, _ := json.Marshal(tc.want)
			if string(gotJSON) != string(wantJSON) {
				t.Fatalf("typed value mismatch:\ngot  %s\nwant %s", gotJSON, wantJSON)
			}
			// MarshalJSON on the wrapper should reproduce the underlying value.
			wrapped, err := json.Marshal(v)
			if err != nil {
				t.Fatalf("marshal wrapper: %v", err)
			}
			if string(wrapped) != string(gotJSON) {
				t.Fatalf("wrapper marshal mismatch:\ngot  %s\nwant %s", wrapped, gotJSON)
			}
		})
	}
}

func TestAgentDefinitionValue_NilMarshalsToNull(t *testing.T) {
	var v AgentDefinitionValue
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(b) != "null" {
		t.Fatalf("got %s want null", b)
	}
}

func TestAgentDefinitionUnion_KindMethods(t *testing.T) {
	cases := []struct {
		v    AgentDefinitionUnion
		want AgentKind
	}{
		{PromptAgentDefinition{}, AgentKindPrompt},
		{HostedAgentDefinition{}, AgentKindHosted},
		{WorkflowAgentDefinition{}, AgentKindWorkflow},
		{AgentDefinition{}, AgentKind("")},
	}
	for _, tc := range cases {
		if got := tc.v.agentKind(); got != tc.want {
			t.Fatalf("%T.agentKind(): got %q want %q", tc.v, got, tc.want)
		}
	}
}
