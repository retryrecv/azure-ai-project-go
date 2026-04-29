package agents

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
	isolation   string
	accept      string
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
		isolation:   req.Header.Get("x-session-isolation-key"),
		accept:      req.Header.Get("Accept"),
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

func TestPatchAgent_PatchesMergeJSON(t *testing.T) {
	st := &scriptedTransport{responses: []scriptedResponse{{
		body: `{"name":"a1"}`,
	}}}
	c := newTestClient(t, st)
	got, err := c.PatchAgent(context.Background(), "a1", PatchAgentBody{
		AgentEndpoint: json.RawMessage(`{"url":"x"}`),
	}, nil)
	if err != nil {
		t.Fatalf("PatchAgent: %v", err)
	}
	if got.Name != "a1" {
		t.Fatalf("decoded: %+v", got)
	}
	call := st.calls[0]
	if call.method != http.MethodPatch || call.path != "/agents/a1" {
		t.Fatalf("method/path: %+v", call)
	}
	if call.contentType != "application/merge-patch+json" {
		t.Fatalf("contentType: %q", call.contentType)
	}
	if call.foundryFeat != hostedAndEndpoint {
		t.Fatalf("foundry: %q", call.foundryFeat)
	}
}

func TestPatchAgent_RequiresName(t *testing.T) {
	c := newTestClient(t, &scriptedTransport{responses: []scriptedResponse{{}}})
	if _, err := c.PatchAgent(context.Background(), "", PatchAgentBody{}, nil); err == nil {
		t.Fatal("expected error")
	}
}

func TestCreateSession_PostsBodyExpects201AndIsolation(t *testing.T) {
	st := &scriptedTransport{responses: []scriptedResponse{{
		status: http.StatusCreated,
		body:   `{"session_id":"s1","agent_session_id":"as1"}`,
	}}}
	c := newTestClient(t, st)
	got, err := c.CreateSession(context.Background(), "a1", "iso-1", CreateSessionBody{
		AgentSessionID:   "as1",
		VersionIndicator: json.RawMessage(`{"type":"latest"}`),
	}, nil)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	if got.SessionID != "s1" {
		t.Fatalf("decoded: %+v", got)
	}
	call := st.calls[0]
	if call.method != http.MethodPost || call.path != "/agents/a1/endpoint/sessions" {
		t.Fatalf("method/path: %+v", call)
	}
	if call.isolation != "iso-1" {
		t.Fatalf("isolation: %q", call.isolation)
	}
	var sent map[string]json.RawMessage
	if err := json.Unmarshal(call.body, &sent); err != nil {
		t.Fatalf("body: %v", err)
	}
	if !strings.Contains(string(sent["version_indicator"]), `"type":"latest"`) {
		t.Fatalf("version_indicator: %s", sent["version_indicator"])
	}
}

func TestCreateSession_Requires(t *testing.T) {
	c := newTestClient(t, &scriptedTransport{responses: []scriptedResponse{{}}})
	if _, err := c.CreateSession(context.Background(), "", "iso", CreateSessionBody{}, nil); err == nil {
		t.Fatal("expected error for empty agentName")
	}
	if _, err := c.CreateSession(context.Background(), "a", "", CreateSessionBody{}, nil); err == nil {
		t.Fatal("expected error for empty isolationKey")
	}
}

func TestGetSession_FiresGet(t *testing.T) {
	st := &scriptedTransport{responses: []scriptedResponse{{
		body: `{"session_id":"s1"}`,
	}}}
	c := newTestClient(t, st)
	got, err := c.GetSession(context.Background(), "a1", "s1", nil)
	if err != nil {
		t.Fatalf("GetSession: %v", err)
	}
	if got.SessionID != "s1" {
		t.Fatalf("decoded: %+v", got)
	}
	call := st.calls[0]
	if call.method != http.MethodGet || call.path != "/agents/a1/endpoint/sessions/s1" {
		t.Fatalf("method/path: %+v", call)
	}
}

func TestDeleteSession_FiresDelete204WithIsolation(t *testing.T) {
	st := &scriptedTransport{responses: []scriptedResponse{{status: http.StatusNoContent}}}
	c := newTestClient(t, st)
	if _, err := c.DeleteSession(context.Background(), "a1", "s1", "iso-2", nil); err != nil {
		t.Fatalf("DeleteSession: %v", err)
	}
	call := st.calls[0]
	if call.method != http.MethodDelete || call.path != "/agents/a1/endpoint/sessions/s1" {
		t.Fatalf("method/path: %+v", call)
	}
	if call.isolation != "iso-2" {
		t.Fatalf("isolation: %q", call.isolation)
	}
}

func TestListSessions_FiresGet(t *testing.T) {
	st := &scriptedTransport{responses: []scriptedResponse{{
		body: `{"data":[{"session_id":"s1"}],"has_more":false}`,
	}}}
	c := newTestClient(t, st)
	limit := int32(15)
	pager := c.NewListSessionsPager("a1", &ListSessionsOptions{Limit: &limit})
	page, err := pager.NextPage(context.Background())
	if err != nil {
		t.Fatalf("NextPage: %v", err)
	}
	if len(page.Data) != 1 {
		t.Fatalf("decoded: %+v", page)
	}
	call := st.calls[0]
	if call.path != "/agents/a1/endpoint/sessions" {
		t.Fatalf("path: %s", call.path)
	}
	if !strings.Contains(call.query, "limit=15") {
		t.Fatalf("query: %s", call.query)
	}
}

func TestListSessions_RequiresAgent(t *testing.T) {
	c := newTestClient(t, &scriptedTransport{responses: []scriptedResponse{{}}})
	pager := c.NewListSessionsPager("", nil)
	if _, err := pager.NextPage(context.Background()); err == nil {
		t.Fatal("expected error")
	}
}

func TestListSessionFiles_FiresGetWithPath(t *testing.T) {
	st := &scriptedTransport{responses: []scriptedResponse{{
		body: `{"path":"/work","entries":[]}`,
	}}}
	c := newTestClient(t, st)
	got, err := c.ListSessionFiles(context.Background(), "a1", "s1", "/work", nil)
	if err != nil {
		t.Fatalf("ListSessionFiles: %v", err)
	}
	if got.Path != "/work" {
		t.Fatalf("decoded: %+v", got)
	}
	call := st.calls[0]
	if call.path != "/agents/a1/endpoint/sessions/s1/files" {
		t.Fatalf("path: %s", call.path)
	}
	if !strings.Contains(call.query, "path=%2Fwork") {
		t.Fatalf("query: %s", call.query)
	}
}

func TestDeleteSessionFile_FiresDelete204WithRecursive(t *testing.T) {
	st := &scriptedTransport{responses: []scriptedResponse{{status: http.StatusNoContent}}}
	c := newTestClient(t, st)
	rec := true
	if _, err := c.DeleteSessionFile(context.Background(), "a1", "s1", "/work/x", &DeleteSessionFileOptions{Recursive: &rec}); err != nil {
		t.Fatalf("DeleteSessionFile: %v", err)
	}
	call := st.calls[0]
	if call.method != http.MethodDelete || call.path != "/agents/a1/endpoint/sessions/s1/files" {
		t.Fatalf("method/path: %+v", call)
	}
	if !strings.Contains(call.query, "recursive=true") {
		t.Fatalf("query: %s", call.query)
	}
}

func TestDownloadSessionFile_FiresGetReturnsBody(t *testing.T) {
	st := &scriptedTransport{responses: []scriptedResponse{{
		body: "RAWBYTES",
	}}}
	c := newTestClient(t, st)
	got, err := c.DownloadSessionFile(context.Background(), "a1", "s1", "/work/x", nil)
	if err != nil {
		t.Fatalf("DownloadSessionFile: %v", err)
	}
	defer got.Body.Close()
	b, err := io.ReadAll(got.Body)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(b) != "RAWBYTES" {
		t.Fatalf("body: %q", b)
	}
	call := st.calls[0]
	if call.method != http.MethodGet || call.path != "/agents/a1/endpoint/sessions/s1/files/content" {
		t.Fatalf("method/path: %+v", call)
	}
	if call.accept != "application/octet-stream" {
		t.Fatalf("accept: %q", call.accept)
	}
}

func TestUploadSessionFile_PutsBinaryAcceptsBoth(t *testing.T) {
	for _, status := range []int{http.StatusOK, http.StatusCreated} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			st := &scriptedTransport{responses: []scriptedResponse{{
				status: status,
				body:   `{"path":"/work/x","size":3}`,
			}}}
			c := newTestClient(t, st)
			got, err := c.UploadSessionFile(context.Background(), "a1", "s1", "/work/x", []byte("abc"), nil)
			if err != nil {
				t.Fatalf("UploadSessionFile: %v", err)
			}
			if got.Size != 3 || got.Path != "/work/x" {
				t.Fatalf("decoded: %+v", got)
			}
			call := st.calls[0]
			if call.method != http.MethodPut || call.path != "/agents/a1/endpoint/sessions/s1/files/content" {
				t.Fatalf("method/path: %+v", call)
			}
			if call.contentType != "application/octet-stream" {
				t.Fatalf("contentType: %q", call.contentType)
			}
			if string(call.body) != "abc" {
				t.Fatalf("body: %q", call.body)
			}
		})
	}
}

func TestUploadSessionFile_RequiresContent(t *testing.T) {
	c := newTestClient(t, &scriptedTransport{responses: []scriptedResponse{{}}})
	if _, err := c.UploadSessionFile(context.Background(), "a", "s", "/p", nil, nil); err == nil {
		t.Fatal("expected error")
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
