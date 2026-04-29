package agents

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
)

func TestCreate_PostsBodyAndDecodes(t *testing.T) {
	st := &scriptedTransport{responses: []scriptedResponse{{
		body: `{"object":"agent","id":"a_1","name":"my-agent","versions":{"latest":{"object":"agent.version","id":"av_1","name":"my-agent","version":"1","created_at":1700000000,"definition":{"kind":"prompt","model":"gpt-5.4"}}}}`,
	}}}
	c := newTestClient(t, st)
	desc := "demo"
	ff := "AgentEndpointsExperimental=Yes"
	got, err := c.Create(context.Background(), "my-agent",
		PromptAgentDefinition{Kind: AgentKindPrompt, Model: "gpt-5.4"},
		&CreateOptions{
			FoundryFeatures: &ff,
			Metadata:        map[string]string{"k": "v"},
			Description:     &desc,
		})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if got.Name != "my-agent" {
		t.Fatalf("decoded agent: %+v", got)
	}
	call := st.calls[0]
	if call.method != http.MethodPost || call.path != "/agents" {
		t.Fatalf("method/path: %+v", call)
	}
	if !strings.Contains(call.query, "api-version=v1") {
		t.Fatalf("query: %s", call.query)
	}
	var body struct {
		Name        string            `json:"name"`
		Metadata    map[string]string `json:"metadata"`
		Description *string           `json:"description"`
		Definition  json.RawMessage   `json:"definition"`
	}
	if err := json.Unmarshal(call.body, &body); err != nil {
		t.Fatalf("body unmarshal: %v", err)
	}
	if body.Name != "my-agent" || body.Description == nil || *body.Description != desc ||
		body.Metadata["k"] != "v" {
		t.Fatalf("body fields: %+v", body)
	}
	if !strings.Contains(string(body.Definition), `"kind":"prompt"`) ||
		!strings.Contains(string(body.Definition), `"model":"gpt-5.4"`) {
		t.Fatalf("definition JSON missing kind/model: %s", body.Definition)
	}
}

func TestCreate_FoundryFeaturesHeaderAppendsSuffix(t *testing.T) {
	st := &scriptedTransport{responses: []scriptedResponse{{
		body: `{"object":"agent","id":"a_1","name":"x","versions":{"latest":{"object":"agent.version","id":"av_1","name":"x","version":"1","created_at":1,"definition":{"kind":"prompt"}}}}`,
	}}}
	// Wrap the scripted transport so we can also inspect headers.
	hc := &headerCapturingTransport{inner: st}
	c, err := NewClient("https://example.test", fakeCred{}, &ClientOptions{
		ClientOptions: azcore.ClientOptions{Transport: hc},
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	ff := "MyFlag=on"
	if _, err := c.Create(context.Background(), "x",
		PromptAgentDefinition{Kind: AgentKindPrompt, Model: "m"},
		&CreateOptions{FoundryFeatures: &ff}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if hc.lastHeader.Get("foundry-features") != "MyFlag=on,AgentEndpoints=V1Preview" {
		t.Fatalf("foundry-features header: %q", hc.lastHeader.Get("foundry-features"))
	}
}

func TestCreate_RequiresArgs(t *testing.T) {
	c := newTestClient(t, &scriptedTransport{responses: []scriptedResponse{{}}})
	if _, err := c.Create(context.Background(), "", PromptAgentDefinition{}, nil); err == nil {
		t.Fatal("expected error for empty name")
	}
	if _, err := c.Create(context.Background(), "x", nil, nil); err == nil {
		t.Fatal("expected error for nil definition")
	}
}

func TestUpdate_PostsToAgentsName(t *testing.T) {
	st := &scriptedTransport{responses: []scriptedResponse{{
		body: `{"object":"agent","id":"a_1","name":"x","versions":{"latest":{"object":"agent.version","id":"av_2","name":"x","version":"2","created_at":1,"definition":{"kind":"workflow","workflow":"w"}}}}`,
	}}}
	c := newTestClient(t, st)
	got, err := c.Update(context.Background(), "x",
		WorkflowAgentDefinition{Kind: AgentKindWorkflow, Workflow: "w"},
		nil)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if got.Versions.Latest.Version != "2" {
		t.Fatalf("decoded: %+v", got)
	}
	call := st.calls[0]
	if call.method != http.MethodPost || call.path != "/agents/x" {
		t.Fatalf("method/path: %+v", call)
	}
	var body struct {
		Definition json.RawMessage `json:"definition"`
	}
	if err := json.Unmarshal(call.body, &body); err != nil {
		t.Fatalf("body unmarshal: %v", err)
	}
	if !strings.Contains(string(body.Definition), `"kind":"workflow"`) {
		t.Fatalf("definition: %s", body.Definition)
	}
	// Update body must NOT include name/agent_endpoint/agent_card.
	if strings.Contains(string(call.body), `"name"`) ||
		strings.Contains(string(call.body), `"agent_endpoint"`) ||
		strings.Contains(string(call.body), `"agent_card"`) {
		t.Fatalf("update body should not include create-only fields: %s", call.body)
	}
}

func TestCreateVersion_PostsToVersionsCollection(t *testing.T) {
	st := &scriptedTransport{responses: []scriptedResponse{{
		body: `{"object":"agent.version","id":"av_3","name":"x","version":"3","created_at":1,"definition":{"kind":"hosted","cpu":"1","memory":"2Gi"}}`,
	}}}
	c := newTestClient(t, st)
	got, err := c.CreateVersion(context.Background(), "x",
		HostedAgentDefinition{Kind: AgentKindHosted, CPU: "1", Memory: "2Gi"},
		nil)
	if err != nil {
		t.Fatalf("CreateVersion: %v", err)
	}
	if got.Version != "3" {
		t.Fatalf("decoded: %+v", got)
	}
	call := st.calls[0]
	if call.method != http.MethodPost || call.path != "/agents/x/versions" {
		t.Fatalf("method/path: %+v", call)
	}
}

func TestDelete_FiresDeleteAndDecodes(t *testing.T) {
	st := &scriptedTransport{responses: []scriptedResponse{{
		body: `{"object":"agent.deleted","name":"x","deleted":true}`,
	}}}
	c := newTestClient(t, st)
	got, err := c.Delete(context.Background(), "x", nil)
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if got.Object != "agent.deleted" || got.Name != "x" || !got.Deleted {
		t.Fatalf("decoded: %+v", got)
	}
	call := st.calls[0]
	if call.method != http.MethodDelete || call.path != "/agents/x" {
		t.Fatalf("method/path: %+v", call)
	}
}

func TestDeleteVersion_FiresDeleteAndDecodes(t *testing.T) {
	st := &scriptedTransport{responses: []scriptedResponse{{
		body: `{"object":"agent.version.deleted","name":"x","version":"3","deleted":true}`,
	}}}
	c := newTestClient(t, st)
	got, err := c.DeleteVersion(context.Background(), "x", "3", nil)
	if err != nil {
		t.Fatalf("DeleteVersion: %v", err)
	}
	if !got.Deleted || got.Version != "3" {
		t.Fatalf("decoded: %+v", got)
	}
	call := st.calls[0]
	if call.method != http.MethodDelete || call.path != "/agents/x/versions/3" {
		t.Fatalf("method/path: %+v", call)
	}
}

func TestDelete_RequiresName(t *testing.T) {
	c := newTestClient(t, &scriptedTransport{responses: []scriptedResponse{{}}})
	if _, err := c.Delete(context.Background(), "", nil); err == nil {
		t.Fatal("expected error for empty agentName")
	}
}

func TestDeleteVersion_RequiresBoth(t *testing.T) {
	c := newTestClient(t, &scriptedTransport{responses: []scriptedResponse{{}}})
	if _, err := c.DeleteVersion(context.Background(), "", "1", nil); err == nil {
		t.Fatal("expected error for empty agentName")
	}
	if _, err := c.DeleteVersion(context.Background(), "x", "", nil); err == nil {
		t.Fatal("expected error for empty agentVersion")
	}
}

// Sanity guard: foundryFeatures defaults to nil header.
func TestUpdate_NoFoundryFeaturesHeaderWhenUnset(t *testing.T) {
	st := &scriptedTransport{responses: []scriptedResponse{{
		body: `{"object":"agent","id":"a_1","name":"x","versions":{"latest":{"object":"agent.version","id":"av_1","name":"x","version":"1","created_at":1,"definition":{"kind":"prompt"}}}}`,
	}}}
	hc := &headerCapturingTransport{inner: st}
	c, err := NewClient("https://example.test", fakeCred{}, &ClientOptions{
		ClientOptions: azcore.ClientOptions{Transport: hc},
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if _, err := c.Update(context.Background(), "x",
		PromptAgentDefinition{Kind: AgentKindPrompt, Model: "m"}, nil); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if v := hc.lastHeader.Get("foundry-features"); v != "" {
		t.Fatalf("foundry-features should be unset: %q", v)
	}
}

// --- helpers used by the foundry-features tests ---

type headerCapturingTransport struct {
	inner      *scriptedTransport
	lastHeader http.Header
}

func (h *headerCapturingTransport) Do(req *http.Request) (*http.Response, error) {
	h.lastHeader = req.Header.Clone()
	return h.inner.Do(req)
}
