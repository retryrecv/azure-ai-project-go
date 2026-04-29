package agents

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestCreateFromManifest_PostsCorrectShape(t *testing.T) {
	st := &scriptedTransport{responses: []scriptedResponse{{
		body: `{"object":"agent","id":"a_1","name":"x","versions":{"latest":{"object":"agent.version","id":"av_1","name":"x","version":"1","created_at":1700000000,"definition":{"kind":"prompt"}}}}`,
	}}}
	c := newTestClient(t, st)
	desc := "from manifest"
	got, err := c.CreateFromManifest(context.Background(), "x", "manifest-42",
		map[string]any{"region": "westus", "size": 3.0},
		&CreateFromManifestOptions{Metadata: map[string]string{"k": "v"}, Description: &desc})
	if err != nil {
		t.Fatalf("CreateFromManifest: %v", err)
	}
	if got.Name != "x" {
		t.Fatalf("decoded: %+v", got)
	}
	call := st.calls[0]
	if call.method != http.MethodPost || call.path != "/agents:import" {
		t.Fatalf("method/path: %+v", call)
	}
	for _, want := range []string{
		`"name":"x"`,
		`"manifest_id":"manifest-42"`,
		`"parameter_values":{`,
		`"region":"westus"`,
		`"size":3`,
		`"description":"from manifest"`,
		`"metadata":{"k":"v"}`,
	} {
		if !strings.Contains(string(call.body), want) {
			t.Fatalf("body missing %s: %s", want, call.body)
		}
	}
}

func TestUpdateFromManifest_PostsCorrectShape(t *testing.T) {
	st := &scriptedTransport{responses: []scriptedResponse{{
		body: `{"object":"agent","id":"a_1","name":"x","versions":{"latest":{"object":"agent.version","id":"av_2","name":"x","version":"2","created_at":1,"definition":{"kind":"prompt"}}}}`,
	}}}
	c := newTestClient(t, st)
	got, err := c.UpdateFromManifest(context.Background(), "x", "m-1",
		map[string]any{"a": "b"}, nil)
	if err != nil {
		t.Fatalf("UpdateFromManifest: %v", err)
	}
	if got.Versions.Latest.Version != "2" {
		t.Fatalf("decoded: %+v", got)
	}
	call := st.calls[0]
	if call.method != http.MethodPost || call.path != "/agents/x/import" {
		t.Fatalf("method/path: %+v", call)
	}
	// No "name" field on the wire for the update-from-manifest variant.
	if strings.Contains(string(call.body), `"name"`) {
		t.Fatalf("update-from-manifest body should not include name: %s", call.body)
	}
	var body map[string]any
	if err := json.Unmarshal(call.body, &body); err != nil {
		t.Fatalf("body unmarshal: %v", err)
	}
	if body["manifest_id"] != "m-1" || body["parameter_values"].(map[string]any)["a"] != "b" {
		t.Fatalf("body: %+v", body)
	}
}

func TestCreateVersionFromManifest_PostsCorrectShape(t *testing.T) {
	st := &scriptedTransport{responses: []scriptedResponse{{
		body: `{"object":"agent.version","id":"av_3","name":"x","version":"3","created_at":1,"definition":{"kind":"prompt"}}`,
	}}}
	c := newTestClient(t, st)
	got, err := c.CreateVersionFromManifest(context.Background(), "x", "m-2",
		map[string]any{}, nil)
	if err != nil {
		t.Fatalf("CreateVersionFromManifest: %v", err)
	}
	if got.Version != "3" {
		t.Fatalf("decoded: %+v", got)
	}
	call := st.calls[0]
	if call.method != http.MethodPost || call.path != "/agents/x/versions:import" {
		t.Fatalf("method/path: %+v", call)
	}
}

func TestManifestOps_RequireArgs(t *testing.T) {
	c := newTestClient(t, &scriptedTransport{responses: []scriptedResponse{{}}})
	if _, err := c.CreateFromManifest(context.Background(), "", "m", nil, nil); err == nil {
		t.Fatal("expected error: empty name")
	}
	if _, err := c.CreateFromManifest(context.Background(), "x", "", nil, nil); err == nil {
		t.Fatal("expected error: empty manifestID")
	}
	if _, err := c.UpdateFromManifest(context.Background(), "", "m", nil, nil); err == nil {
		t.Fatal("expected error: empty agentName")
	}
	if _, err := c.UpdateFromManifest(context.Background(), "x", "", nil, nil); err == nil {
		t.Fatal("expected error: empty manifestID")
	}
	if _, err := c.CreateVersionFromManifest(context.Background(), "", "m", nil, nil); err == nil {
		t.Fatal("expected error: empty agentName")
	}
	if _, err := c.CreateVersionFromManifest(context.Background(), "x", "", nil, nil); err == nil {
		t.Fatal("expected error: empty manifestID")
	}
}

func TestManifestOps_NilParameterValuesEncodesEmptyObject(t *testing.T) {
	st := &scriptedTransport{responses: []scriptedResponse{{
		body: `{"object":"agent","id":"a_1","name":"x","versions":{"latest":{"object":"agent.version","id":"av_1","name":"x","version":"1","created_at":1,"definition":{"kind":"prompt"}}}}`,
	}}}
	c := newTestClient(t, st)
	if _, err := c.CreateFromManifest(context.Background(), "x", "m", nil, nil); err != nil {
		t.Fatalf("CreateFromManifest: %v", err)
	}
	if !strings.Contains(string(st.calls[0].body), `"parameter_values":{}`) {
		t.Fatalf("expected parameter_values:{} in body: %s", st.calls[0].body)
	}
}
