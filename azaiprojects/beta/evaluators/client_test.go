package evaluators

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

func TestList_FiresGetWithFilters(t *testing.T) {
	st := &scriptedTransport{responses: []scriptedResponse{{
		body: `{"value":[{"name":"e1","version":"1","evaluator_type":"builtin"}]}`,
	}}}
	c := newTestClient(t, st)
	ty := "builtin"
	limit := int32(10)
	pager := c.NewListPager(&ListOptions{EvaluatorType: &ty, Limit: &limit})
	page, err := pager.NextPage(context.Background())
	if err != nil {
		t.Fatalf("NextPage: %v", err)
	}
	if len(page.Value) != 1 || page.Value[0].Name != "e1" {
		t.Fatalf("decoded: %+v", page)
	}
	call := st.calls[0]
	if call.method != http.MethodGet || call.path != "/evaluators" {
		t.Fatalf("method/path: %+v", call)
	}
	for _, want := range []string{"type=builtin", "limit=10", "api-version=v1"} {
		if !strings.Contains(call.query, want) {
			t.Fatalf("missing %s in %s", want, call.query)
		}
	}
	if call.foundryFeat != foundryHeader {
		t.Fatalf("foundry: %q", call.foundryFeat)
	}
}

func TestListVersions_FiresGet(t *testing.T) {
	st := &scriptedTransport{responses: []scriptedResponse{{
		body: `{"value":[{"name":"e1","version":"2"}]}`,
	}}}
	c := newTestClient(t, st)
	pager := c.NewListVersionsPager("e1", nil)
	page, err := pager.NextPage(context.Background())
	if err != nil {
		t.Fatalf("NextPage: %v", err)
	}
	if len(page.Value) != 1 || page.Value[0].Version != "2" {
		t.Fatalf("decoded: %+v", page)
	}
	if st.calls[0].path != "/evaluators/e1/versions" {
		t.Fatalf("path: %s", st.calls[0].path)
	}
}

func TestListVersions_RequiresName(t *testing.T) {
	c := newTestClient(t, &scriptedTransport{responses: []scriptedResponse{{}}})
	pager := c.NewListVersionsPager("", nil)
	if _, err := pager.NextPage(context.Background()); err == nil {
		t.Fatal("expected error")
	}
}

func TestGetVersion_FiresGet(t *testing.T) {
	st := &scriptedTransport{responses: []scriptedResponse{{
		body: `{"name":"e1","version":"3","display_name":"d"}`,
	}}}
	c := newTestClient(t, st)
	got, err := c.GetVersion(context.Background(), "e1", "3", nil)
	if err != nil {
		t.Fatalf("GetVersion: %v", err)
	}
	if got.Version != "3" || got.DisplayName != "d" {
		t.Fatalf("decoded: %+v", got)
	}
	call := st.calls[0]
	if call.method != http.MethodGet || call.path != "/evaluators/e1/versions/3" {
		t.Fatalf("method/path: %+v", call)
	}
}

func TestGetVersion_Requires(t *testing.T) {
	c := newTestClient(t, &scriptedTransport{responses: []scriptedResponse{{}}})
	if _, err := c.GetVersion(context.Background(), "", "v", nil); err == nil {
		t.Fatal("expected error for empty name")
	}
	if _, err := c.GetVersion(context.Background(), "e", "", nil); err == nil {
		t.Fatal("expected error for empty version")
	}
}

func TestCreateVersion_PostsBodyExpects201(t *testing.T) {
	st := &scriptedTransport{responses: []scriptedResponse{{
		status: http.StatusCreated,
		body:   `{"name":"e1","version":"1","display_name":"D"}`,
	}}}
	c := newTestClient(t, st)
	got, err := c.CreateVersion(context.Background(), "e1", EvaluatorVersion{
		DisplayName:   "D",
		EvaluatorType: "custom",
		Categories:    []string{"quality"},
		Definition:    json.RawMessage(`{"type":"code","init_parameters":{}}`),
	}, nil)
	if err != nil {
		t.Fatalf("CreateVersion: %v", err)
	}
	if got.DisplayName != "D" {
		t.Fatalf("decoded: %+v", got)
	}
	call := st.calls[0]
	if call.method != http.MethodPost || call.path != "/evaluators/e1/versions" {
		t.Fatalf("method/path: %+v", call)
	}
	var sent map[string]json.RawMessage
	if err := json.Unmarshal(call.body, &sent); err != nil {
		t.Fatalf("body: %v", err)
	}
	if !strings.Contains(string(sent["definition"]), `"type":"code"`) {
		t.Fatalf("definition: %s", sent["definition"])
	}
}

func TestCreateVersion_RequiresName(t *testing.T) {
	c := newTestClient(t, &scriptedTransport{responses: []scriptedResponse{{}}})
	if _, err := c.CreateVersion(context.Background(), "", EvaluatorVersion{}, nil); err == nil {
		t.Fatal("expected error")
	}
}

func TestUpdateVersion_PatchesBody(t *testing.T) {
	st := &scriptedTransport{responses: []scriptedResponse{{
		body: `{"name":"e1","version":"1","display_name":"D2"}`,
	}}}
	c := newTestClient(t, st)
	got, err := c.UpdateVersion(context.Background(), "e1", "1", EvaluatorVersion{DisplayName: "D2"}, nil)
	if err != nil {
		t.Fatalf("UpdateVersion: %v", err)
	}
	if got.DisplayName != "D2" {
		t.Fatalf("decoded: %+v", got)
	}
	call := st.calls[0]
	if call.method != http.MethodPatch || call.path != "/evaluators/e1/versions/1" {
		t.Fatalf("method/path: %+v", call)
	}
}

func TestUpdateVersion_Requires(t *testing.T) {
	c := newTestClient(t, &scriptedTransport{responses: []scriptedResponse{{}}})
	if _, err := c.UpdateVersion(context.Background(), "", "v", EvaluatorVersion{}, nil); err == nil {
		t.Fatal("expected error for empty name")
	}
	if _, err := c.UpdateVersion(context.Background(), "e", "", EvaluatorVersion{}, nil); err == nil {
		t.Fatal("expected error for empty version")
	}
}

func TestDeleteVersion_FiresDelete204(t *testing.T) {
	st := &scriptedTransport{responses: []scriptedResponse{{status: http.StatusNoContent}}}
	c := newTestClient(t, st)
	if _, err := c.DeleteVersion(context.Background(), "e1", "1", nil); err != nil {
		t.Fatalf("DeleteVersion: %v", err)
	}
	call := st.calls[0]
	if call.method != http.MethodDelete || call.path != "/evaluators/e1/versions/1" {
		t.Fatalf("method/path: %+v", call)
	}
}

func TestDeleteVersion_Requires(t *testing.T) {
	c := newTestClient(t, &scriptedTransport{responses: []scriptedResponse{{}}})
	if _, err := c.DeleteVersion(context.Background(), "", "v", nil); err == nil {
		t.Fatal("expected error for empty name")
	}
	if _, err := c.DeleteVersion(context.Background(), "e", "", nil); err == nil {
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
