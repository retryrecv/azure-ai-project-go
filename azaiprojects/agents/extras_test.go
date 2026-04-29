package agents

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestAgentEndpoint_RoundTrip(t *testing.T) {
	in := AgentEndpoint{
		Protocols:            []AgentEndpointProtocol{AgentEndpointProtocolActivity, AgentEndpointProtocolA2A},
		VersionSelector:      json.RawMessage(`{"version_selection_rules":[{"type":"FixedRatio","agent_version":"1","traffic_percentage":100}]}`),
		AuthorizationSchemes: []json.RawMessage{json.RawMessage(`{"type":"BotService"}`)},
	}
	b, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	for _, want := range []string{`"protocols":["activity","a2a"]`, `"version_selector":{`, `"authorization_schemes":[{`} {
		if !strings.Contains(string(b), want) {
			t.Fatalf("missing %s in JSON: %s", want, b)
		}
	}
	var out AgentEndpoint
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(out.Protocols) != 2 || out.Protocols[0] != AgentEndpointProtocolActivity ||
		len(out.AuthorizationSchemes) != 1 || len(out.VersionSelector) == 0 {
		t.Fatalf("decoded mismatch: %+v", out)
	}
}

func TestAgentCard_RoundTrip(t *testing.T) {
	in := AgentCard{
		Version:     "1.0",
		Description: "A demo agent",
		Skills: []AgentCardSkill{{
			ID:          "summarize",
			Name:        "Summarize",
			Description: "Summarize text",
			Tags:        []string{"text", "nlp"},
			Examples:    []string{"summarize this article"},
		}},
	}
	b, _ := json.Marshal(in)
	if !strings.Contains(string(b), `"version":"1.0"`) ||
		!strings.Contains(string(b), `"skills":[`) ||
		!strings.Contains(string(b), `"id":"summarize"`) {
		t.Fatalf("unexpected JSON: %s", b)
	}
	var out AgentCard
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.Version != in.Version || len(out.Skills) != 1 ||
		out.Skills[0].ID != "summarize" || out.Skills[0].Tags[0] != "text" {
		t.Fatalf("decoded mismatch: %+v", out)
	}
}

func TestAgentBlueprintReference_RoundTrip(t *testing.T) {
	in := AgentBlueprintReference{
		Type:        AgentBlueprintReferenceTypeManagedAgentIdentity,
		BlueprintID: "bp_42",
	}
	b, _ := json.Marshal(in)
	want := `{"type":"ManagedAgentIdentityBlueprint","blueprint_id":"bp_42"}`
	if string(b) != want {
		t.Fatalf("got %s want %s", b, want)
	}
	var out AgentBlueprintReference
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out != in {
		t.Fatalf("mismatch: %+v", out)
	}
}

func TestDeleteResponses_RoundTrip(t *testing.T) {
	a := DeleteAgentResponse{Object: "agent.deleted", Name: "x", Deleted: true}
	ba, _ := json.Marshal(a)
	if string(ba) != `{"object":"agent.deleted","name":"x","deleted":true}` {
		t.Fatalf("agent: %s", ba)
	}
	v := DeleteAgentVersionResponse{Object: "agent.version.deleted", Name: "x", Version: "1", Deleted: true}
	bv, _ := json.Marshal(v)
	if string(bv) != `{"object":"agent.version.deleted","name":"x","version":"1","deleted":true}` {
		t.Fatalf("version: %s", bv)
	}
	var ad DeleteAgentResponse
	if err := json.Unmarshal(ba, &ad); err != nil || ad != a {
		t.Fatalf("agent round-trip: %v %+v", err, ad)
	}
	var vd DeleteAgentVersionResponse
	if err := json.Unmarshal(bv, &vd); err != nil || vd != v {
		t.Fatalf("version round-trip: %v %+v", err, vd)
	}
}

func TestAgentsPage_DecodesPagination(t *testing.T) {
	raw := []byte(`{
		"data":[{"object":"agent","id":"a_1","name":"x","versions":{"latest":{"object":"agent.version","id":"av_1","name":"x","version":"1","created_at":1700000000,"definition":{"kind":"prompt"}}}}],
		"first_id":"a_1",
		"last_id":"a_1",
		"has_more":false
	}`)
	var p AgentsPage
	if err := json.Unmarshal(raw, &p); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(p.Data) != 1 || p.Data[0].ID != "a_1" || p.HasMore || p.FirstID != "a_1" || p.LastID != "a_1" {
		t.Fatalf("decoded mismatch: %+v", p)
	}
}

func TestAgentVersionsPage_DecodesPagination(t *testing.T) {
	raw := []byte(`{
		"data":[{"object":"agent.version","id":"av_1","name":"x","version":"1","created_at":1700000000,"definition":{"kind":"prompt"}}],
		"has_more":true,
		"last_id":"av_1"
	}`)
	var p AgentVersionsPage
	if err := json.Unmarshal(raw, &p); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(p.Data) != 1 || !p.HasMore || p.LastID != "av_1" {
		t.Fatalf("decoded mismatch: %+v", p)
	}
}

func TestManifestBodies_Marshal(t *testing.T) {
	b1, _ := json.Marshal(agentManifestBody{
		Name: "n", Description: "d",
		ManifestID: "m_1", ParameterValues: map[string]any{"k": "v"},
	})
	for _, want := range []string{`"name":"n"`, `"description":"d"`, `"manifest_id":"m_1"`, `"parameter_values":{"k":"v"}`} {
		if !strings.Contains(string(b1), want) {
			t.Fatalf("create-from-manifest body missing %s: %s", want, b1)
		}
	}
	b2, _ := json.Marshal(agentVersionManifestBody{
		Metadata: map[string]string{"a": "b"}, ManifestID: "m_2",
		ParameterValues: map[string]any{"x": 1.0},
	})
	if strings.Contains(string(b2), `"name"`) {
		t.Fatalf("version-manifest body should not have name: %s", b2)
	}
	for _, want := range []string{`"metadata":{"a":"b"}`, `"manifest_id":"m_2"`, `"parameter_values":{"x":1}`} {
		if !strings.Contains(string(b2), want) {
			t.Fatalf("version-manifest body missing %s: %s", want, b2)
		}
	}
}
