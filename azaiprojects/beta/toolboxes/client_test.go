package toolboxes

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

func TestList_FiresGetWithCursorParams(t *testing.T) {
	st := &scriptedTransport{responses: []scriptedResponse{{
		body: `{"data":[{"id":"t1","name":"tb1","default_version":"1"}],"last_id":"t1","has_more":false}`,
	}}}
	c := newTestClient(t, st)
	limit := int32(10)
	pager := c.NewListPager(&ListOptions{Limit: &limit})
	page, err := pager.NextPage(context.Background())
	if err != nil {
		t.Fatalf("NextPage: %v", err)
	}
	if len(page.Data) != 1 || page.Data[0].Name != "tb1" {
		t.Fatalf("decoded: %+v", page)
	}
	got := st.calls[0]
	if got.method != http.MethodGet || got.path != "/toolboxes" {
		t.Fatalf("method/path: %+v", got)
	}
	for _, want := range []string{"limit=10", "api-version=v1"} {
		if !strings.Contains(got.query, want) {
			t.Fatalf("missing %s in %s", want, got.query)
		}
	}
	if got.foundryFeat != foundryHeader {
		t.Fatalf("foundry: %q", got.foundryFeat)
	}
}

func TestListVersions_FiresGetForToolbox(t *testing.T) {
	st := &scriptedTransport{responses: []scriptedResponse{{
		body: `{"data":[{"id":"v1","name":"tb1","version":"1","created_at":1700000000,"tools":[{"type":"foo"}],"metadata":{}}],"last_id":"v1","has_more":false}`,
	}}}
	c := newTestClient(t, st)
	pager := c.NewListVersionsPager("tb1", nil)
	page, err := pager.NextPage(context.Background())
	if err != nil {
		t.Fatalf("NextPage: %v", err)
	}
	if len(page.Data) != 1 || page.Data[0].Version != "1" {
		t.Fatalf("decoded: %+v", page)
	}
	if len(page.Data[0].Tools) != 1 {
		t.Fatalf("tools: %+v", page.Data[0].Tools)
	}
	if page.Data[0].CreatedAt.Unix() != 1700000000 {
		t.Fatalf("created_at: %v", page.Data[0].CreatedAt)
	}
	call := st.calls[0]
	if call.path != "/toolboxes/tb1/versions" {
		t.Fatalf("path: %s", call.path)
	}
}

func TestListVersions_RequiresName(t *testing.T) {
	c := newTestClient(t, &scriptedTransport{responses: []scriptedResponse{{}}})
	pager := c.NewListVersionsPager("", nil)
	if _, err := pager.NextPage(context.Background()); err == nil {
		t.Fatal("expected error")
	}
}

func TestGet_FiresGetWithName(t *testing.T) {
	st := &scriptedTransport{responses: []scriptedResponse{{
		body: `{"id":"t1","name":"tb1","default_version":"3"}`,
	}}}
	c := newTestClient(t, st)
	got, err := c.Get(context.Background(), "tb1", nil)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.DefaultVersion != "3" {
		t.Fatalf("decoded: %+v", got)
	}
	call := st.calls[0]
	if call.method != http.MethodGet || call.path != "/toolboxes/tb1" {
		t.Fatalf("method/path: %+v", call)
	}
}

func TestGetVersion_FiresGetWithNameAndVersion(t *testing.T) {
	st := &scriptedTransport{responses: []scriptedResponse{{
		body: `{"id":"v1","name":"tb1","version":"2","created_at":1700000000,"tools":[]}`,
	}}}
	c := newTestClient(t, st)
	got, err := c.GetVersion(context.Background(), "tb1", "2", nil)
	if err != nil {
		t.Fatalf("GetVersion: %v", err)
	}
	if got.Version != "2" {
		t.Fatalf("decoded: %+v", got)
	}
	call := st.calls[0]
	if call.path != "/toolboxes/tb1/versions/2" {
		t.Fatalf("path: %s", call.path)
	}
}

func TestGetVersion_Requires(t *testing.T) {
	c := newTestClient(t, &scriptedTransport{responses: []scriptedResponse{{}}})
	if _, err := c.GetVersion(context.Background(), "", "v", nil); err == nil {
		t.Fatal("expected error for empty name")
	}
	if _, err := c.GetVersion(context.Background(), "n", "", nil); err == nil {
		t.Fatal("expected error for empty version")
	}
}

func TestUpdate_PatchesDefaultVersion(t *testing.T) {
	st := &scriptedTransport{responses: []scriptedResponse{{
		body: `{"id":"t1","name":"tb1","default_version":"5"}`,
	}}}
	c := newTestClient(t, st)
	got, err := c.Update(context.Background(), "tb1", "5", nil)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if got.DefaultVersion != "5" {
		t.Fatalf("decoded: %+v", got)
	}
	call := st.calls[0]
	if call.method != http.MethodPatch || call.path != "/toolboxes/tb1" {
		t.Fatalf("method/path: %+v", call)
	}
	var sent UpdateBody
	if err := json.Unmarshal(call.body, &sent); err != nil {
		t.Fatalf("body: %v", err)
	}
	if sent.DefaultVersion != "5" {
		t.Fatalf("body fields: %+v", sent)
	}
}

func TestUpdate_Requires(t *testing.T) {
	c := newTestClient(t, &scriptedTransport{responses: []scriptedResponse{{}}})
	if _, err := c.Update(context.Background(), "", "v", nil); err == nil {
		t.Fatal("expected error for empty name")
	}
	if _, err := c.Update(context.Background(), "n", "", nil); err == nil {
		t.Fatal("expected error for empty version")
	}
}

func TestCreateVersion_PostsToolsBody(t *testing.T) {
	st := &scriptedTransport{responses: []scriptedResponse{{
		body: `{"id":"v1","name":"tb1","version":"3","created_at":1700000000,"tools":[{"type":"echo"}]}`,
	}}}
	c := newTestClient(t, st)
	tools := []json.RawMessage{json.RawMessage(`{"type":"echo"}`)}
	got, err := c.CreateVersion(context.Background(), "tb1", CreateVersionBody{
		Description: "v3", Tools: tools,
	}, nil)
	if err != nil {
		t.Fatalf("CreateVersion: %v", err)
	}
	if got.Version != "3" {
		t.Fatalf("decoded: %+v", got)
	}
	call := st.calls[0]
	if call.method != http.MethodPost || call.path != "/toolboxes/tb1/versions" {
		t.Fatalf("method/path: %+v", call)
	}
	var sent map[string]any
	if err := json.Unmarshal(call.body, &sent); err != nil {
		t.Fatalf("body: %v", err)
	}
	if sent["description"] != "v3" {
		t.Fatalf("body: %v", sent)
	}
	toolsArr, ok := sent["tools"].([]any)
	if !ok || len(toolsArr) != 1 {
		t.Fatalf("tools: %v", sent["tools"])
	}
}

func TestCreateVersion_RequiresName(t *testing.T) {
	c := newTestClient(t, &scriptedTransport{responses: []scriptedResponse{{}}})
	if _, err := c.CreateVersion(context.Background(), "", CreateVersionBody{}, nil); err == nil {
		t.Fatal("expected error")
	}
}

func TestDelete_FiresDelete(t *testing.T) {
	st := &scriptedTransport{responses: []scriptedResponse{{status: http.StatusNoContent}}}
	c := newTestClient(t, st)
	if _, err := c.Delete(context.Background(), "tb1", nil); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	call := st.calls[0]
	if call.method != http.MethodDelete || call.path != "/toolboxes/tb1" {
		t.Fatalf("method/path: %+v", call)
	}
}

func TestDelete_RequiresName(t *testing.T) {
	c := newTestClient(t, &scriptedTransport{responses: []scriptedResponse{{}}})
	if _, err := c.Delete(context.Background(), "", nil); err == nil {
		t.Fatal("expected error")
	}
}

func TestDeleteVersion_FiresDelete(t *testing.T) {
	st := &scriptedTransport{responses: []scriptedResponse{{status: http.StatusNoContent}}}
	c := newTestClient(t, st)
	if _, err := c.DeleteVersion(context.Background(), "tb1", "2", nil); err != nil {
		t.Fatalf("DeleteVersion: %v", err)
	}
	call := st.calls[0]
	if call.method != http.MethodDelete || call.path != "/toolboxes/tb1/versions/2" {
		t.Fatalf("method/path: %+v", call)
	}
}

func TestDeleteVersion_Requires(t *testing.T) {
	c := newTestClient(t, &scriptedTransport{responses: []scriptedResponse{{}}})
	if _, err := c.DeleteVersion(context.Background(), "", "v", nil); err == nil {
		t.Fatal("expected error for empty name")
	}
	if _, err := c.DeleteVersion(context.Background(), "n", "", nil); err == nil {
		t.Fatal("expected error for empty version")
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
