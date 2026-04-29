package redteams

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
)

type fakeCred struct{}

func (fakeCred) GetToken(_ context.Context, _ policy.TokenRequestOptions) (azcore.AccessToken, error) {
	return azcore.AccessToken{Token: "fake", ExpiresOn: time.Now().Add(time.Hour)}, nil
}

type scriptedTransport struct {
	responses []scriptedResponse
	calls     []recordedCall
	idx       int
}

type scriptedResponse struct {
	status int
	body   string
}

type recordedCall struct {
	method      string
	path        string
	query       string
	body        []byte
	contentType string
	foundryFeat string
}

func (s *scriptedTransport) Do(req *http.Request) (*http.Response, error) {
	var body []byte
	if req.Body != nil {
		body, _ = io.ReadAll(req.Body)
	}
	s.calls = append(s.calls, recordedCall{
		method:      req.Method,
		path:        req.URL.Path,
		query:       req.URL.RawQuery,
		body:        body,
		contentType: req.Header.Get("Content-Type"),
		foundryFeat: req.Header.Get("foundry-features"),
	})
	resp := s.responses[s.idx]
	if s.idx < len(s.responses)-1 {
		s.idx++
	}
	status := resp.status
	if status == 0 {
		status = http.StatusOK
	}
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(bytes.NewBufferString(resp.body)),
		Header:     make(http.Header),
		Request:    req,
	}, nil
}

func newTestClient(t *testing.T, st *scriptedTransport) *Client {
	t.Helper()
	c, err := NewClient("https://example.test", fakeCred{}, &ClientOptions{
		ClientOptions: azcore.ClientOptions{Transport: st},
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return c
}

func TestList_FiresGet(t *testing.T) {
	st := &scriptedTransport{responses: []scriptedResponse{{
		body: `{"value":[{"id":"r1","displayName":"demo","status":"Running"}]}`,
	}}}
	c := newTestClient(t, st)
	pager := c.NewListPager(nil)
	page, err := pager.NextPage(context.Background())
	if err != nil {
		t.Fatalf("NextPage: %v", err)
	}
	if len(page.Value) != 1 || page.Value[0].Name != "r1" {
		t.Fatalf("decoded: %+v", page)
	}
	call := st.calls[0]
	if call.method != http.MethodGet || call.path != "/redTeams/runs" {
		t.Fatalf("method/path: %+v", call)
	}
	if !strings.Contains(call.query, "api-version=v1") {
		t.Fatalf("query: %s", call.query)
	}
	if call.foundryFeat != foundryHeader {
		t.Fatalf("foundry: %q", call.foundryFeat)
	}
}

func TestGet_FiresGet(t *testing.T) {
	st := &scriptedTransport{responses: []scriptedResponse{{
		body: `{"id":"r1","displayName":"demo","status":"Done"}`,
	}}}
	c := newTestClient(t, st)
	got, err := c.Get(context.Background(), "r1", nil)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Name != "r1" || got.DisplayName != "demo" {
		t.Fatalf("decoded: %+v", got)
	}
	call := st.calls[0]
	if call.method != http.MethodGet || call.path != "/redTeams/runs/r1" {
		t.Fatalf("method/path: %+v", call)
	}
}

func TestGet_RequiresName(t *testing.T) {
	c := newTestClient(t, &scriptedTransport{responses: []scriptedResponse{{}}})
	if _, err := c.Get(context.Background(), "", nil); err == nil {
		t.Fatal("expected error")
	}
}

func TestCreate_PostsBodyExpects201(t *testing.T) {
	st := &scriptedTransport{responses: []scriptedResponse{{
		status: http.StatusCreated,
		body:   `{"id":"r1","displayName":"demo","status":"Queued"}`,
	}}}
	c := newTestClient(t, st)
	turns := int32(3)
	got, err := c.Create(context.Background(), RedTeam{
		DisplayName: "demo", NumTurns: &turns,
		Target: json.RawMessage(`{"type":"Agent","agentName":"a"}`),
	}, nil)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if got.Name != "r1" {
		t.Fatalf("decoded: %+v", got)
	}
	call := st.calls[0]
	if call.method != http.MethodPost || call.path != "/redTeams/runs:run" {
		t.Fatalf("method/path: %+v", call)
	}
	var sent map[string]any
	if err := json.Unmarshal(call.body, &sent); err != nil {
		t.Fatalf("body: %v", err)
	}
	if sent["displayName"] != "demo" {
		t.Fatalf("body: %v", sent)
	}
	target, _ := sent["target"].(map[string]any)
	if target == nil || target["type"] != "Agent" {
		t.Fatalf("target: %v", sent["target"])
	}
}

func TestNewClient_Validation(t *testing.T) {
	if _, err := NewClient("", fakeCred{}, nil); err == nil {
		t.Fatal("empty endpoint should error")
	}
	if _, err := NewClient("https://example.test", nil, nil); err == nil {
		t.Fatal("nil cred should error")
	}
}
