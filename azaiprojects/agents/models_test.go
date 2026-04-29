package agents

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestRaiConfig_RoundTrip(t *testing.T) {
	in := RaiConfig{RaiPolicyName: "default"}
	b, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if got := string(b); got != `{"rai_policy_name":"default"}` {
		t.Fatalf("unexpected JSON: %s", got)
	}
	var out RaiConfig
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out != in {
		t.Fatalf("round-trip mismatch: got %+v want %+v", out, in)
	}
}

func TestAgentIdentity_RoundTrip(t *testing.T) {
	in := AgentIdentity{PrincipalID: "p-1", ClientID: "c-1"}
	b, _ := json.Marshal(in)
	if !strings.Contains(string(b), `"principal_id":"p-1"`) ||
		!strings.Contains(string(b), `"client_id":"c-1"`) {
		t.Fatalf("unexpected JSON: %s", b)
	}
	var out AgentIdentity
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out != in {
		t.Fatalf("round-trip mismatch: got %+v want %+v", out, in)
	}
}

func TestAgentDefinition_BaseRoundTrip(t *testing.T) {
	in := AgentDefinition{Kind: AgentKindPrompt, RaiConfig: &RaiConfig{RaiPolicyName: "p"}}
	b, _ := json.Marshal(in)
	want := `{"kind":"prompt","rai_config":{"rai_policy_name":"p"}}`
	if string(b) != want {
		t.Fatalf("got %s want %s", b, want)
	}
	var out AgentDefinition
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.Kind != in.Kind || out.RaiConfig == nil || *out.RaiConfig != *in.RaiConfig {
		t.Fatalf("round-trip mismatch: %+v", out)
	}
}

func TestUnixSeconds_MarshalUnmarshal(t *testing.T) {
	ts := UnixSeconds{Time: time.Unix(1700000000, 0).UTC()}
	b, err := json.Marshal(ts)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(b) != "1700000000" {
		t.Fatalf("got %s want 1700000000", b)
	}
	var out UnixSeconds
	if err := json.Unmarshal([]byte("1700000000"), &out); err != nil {
		t.Fatalf("unmarshal int: %v", err)
	}
	if !out.Time.Equal(ts.Time) {
		t.Fatalf("int round-trip mismatch: got %v want %v", out.Time, ts.Time)
	}
	// Float (with sub-second precision) also accepted.
	if err := json.Unmarshal([]byte("1700000000.5"), &out); err != nil {
		t.Fatalf("unmarshal float: %v", err)
	}
	if out.Time.Unix() != 1700000000 {
		t.Fatalf("float seconds wrong: %v", out.Time)
	}
	// Null -> zero.
	if err := json.Unmarshal([]byte("null"), &out); err != nil {
		t.Fatalf("unmarshal null: %v", err)
	}
	if !out.Time.IsZero() {
		t.Fatalf("null should yield zero time, got %v", out.Time)
	}
}

func TestAgentVersion_RoundTrip(t *testing.T) {
	raw := []byte(`{
		"object":"agent.version",
		"id":"av_123",
		"name":"my-agent",
		"version":"1",
		"description":"hi",
		"metadata":{"k":"v"},
		"created_at":1700000000,
		"definition":{"kind":"prompt","rai_config":{"rai_policy_name":"p"}},
		"status":"active",
		"agent_guid":"guid-1"
	}`)
	var v AgentVersion
	if err := json.Unmarshal(raw, &v); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if v.Object != "agent.version" || v.ID != "av_123" || v.Name != "my-agent" ||
		v.Version != "1" || v.Description != "hi" || v.Metadata["k"] != "v" ||
		v.Status != AgentVersionStatusActive || v.AgentGUID != "guid-1" ||
		v.CreatedAt.Time.Unix() != 1700000000 {
		t.Fatalf("decoded fields mismatch: %+v", v)
	}
	if len(v.Definition) == 0 {
		t.Fatalf("definition raw payload missing")
	}
	// Round-trip back to JSON and reparse to confirm field names survive.
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var v2 AgentVersion
	if err := json.Unmarshal(b, &v2); err != nil {
		t.Fatalf("re-unmarshal: %v", err)
	}
	if v2.ID != v.ID || v2.Status != v.Status || v2.CreatedAt.Time.Unix() != v.CreatedAt.Time.Unix() {
		t.Fatalf("round-trip lost fields: %+v vs %+v", v2, v)
	}
}

func TestAgent_RoundTrip(t *testing.T) {
	raw := []byte(`{
		"object":"agent",
		"id":"a_1",
		"name":"my-agent",
		"versions":{
			"latest":{
				"object":"agent.version",
				"id":"av_1",
				"name":"my-agent",
				"version":"3",
				"created_at":1700000000,
				"definition":{"kind":"hosted"}
			}
		}
	}`)
	var a Agent
	if err := json.Unmarshal(raw, &a); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if a.ID != "a_1" || a.Name != "my-agent" || a.Versions.Latest.Version != "3" {
		t.Fatalf("decoded fields mismatch: %+v", a)
	}
	b, err := json.Marshal(a)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var a2 Agent
	if err := json.Unmarshal(b, &a2); err != nil {
		t.Fatalf("re-unmarshal: %v", err)
	}
	if a2.Versions.Latest.ID != a.Versions.Latest.ID {
		t.Fatalf("nested version lost in round-trip")
	}
}
