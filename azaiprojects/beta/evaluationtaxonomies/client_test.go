package evaluationtaxonomies

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
	clientReq   string
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
		clientReq:   req.Header.Get("x-ms-client-request-id"),
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
		body: `{"value":[{"name":"t1","version":"1"}]}`,
	}}}
	c := newTestClient(t, st)
	in := "agent"
	pager := c.NewListPager(&ListOptions{InputType: &in, ClientRequestID: "req-1"})
	page, err := pager.NextPage(context.Background())
	if err != nil {
		t.Fatalf("NextPage: %v", err)
	}
	if len(page.Value) != 1 || page.Value[0].Name != "t1" {
		t.Fatalf("decoded: %+v", page)
	}
	call := st.calls[0]
	if call.method != http.MethodGet || call.path != "/evaluationtaxonomies" {
		t.Fatalf("method/path: %+v", call)
	}
	if !strings.Contains(call.query, "inputType=agent") {
		t.Fatalf("query: %s", call.query)
	}
	if call.clientReq != "req-1" {
		t.Fatalf("client-request-id: %q", call.clientReq)
	}
	if call.foundryFeat != foundryHeader {
		t.Fatalf("foundry: %q", call.foundryFeat)
	}
}

func TestGet_FiresGet(t *testing.T) {
	st := &scriptedTransport{responses: []scriptedResponse{{
		body: `{"name":"t1","version":"1","description":"d"}`,
	}}}
	c := newTestClient(t, st)
	got, err := c.Get(context.Background(), "t1", nil)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Description != "d" {
		t.Fatalf("decoded: %+v", got)
	}
	call := st.calls[0]
	if call.method != http.MethodGet || call.path != "/evaluationtaxonomies/t1" {
		t.Fatalf("method/path: %+v", call)
	}
}

func TestGet_RequiresName(t *testing.T) {
	c := newTestClient(t, &scriptedTransport{responses: []scriptedResponse{{}}})
	if _, err := c.Get(context.Background(), "", nil); err == nil {
		t.Fatal("expected error")
	}
}

func TestCreate_PutsBodyAcceptsBoth200And201(t *testing.T) {
	for _, status := range []int{http.StatusOK, http.StatusCreated} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			st := &scriptedTransport{responses: []scriptedResponse{{
				status: status,
				body:   `{"name":"t1","version":"1"}`,
			}}}
			c := newTestClient(t, st)
			got, err := c.Create(context.Background(), "t1", EvaluationTaxonomy{
				Description:   "x",
				TaxonomyInput: json.RawMessage(`{"type":"agent"}`),
			}, nil)
			if err != nil {
				t.Fatalf("Create: %v", err)
			}
			if got.Name != "t1" {
				t.Fatalf("decoded: %+v", got)
			}
			call := st.calls[0]
			if call.method != http.MethodPut || call.path != "/evaluationtaxonomies/t1" {
				t.Fatalf("method/path: %+v", call)
			}
			var sent map[string]json.RawMessage
			if err := json.Unmarshal(call.body, &sent); err != nil {
				t.Fatalf("body: %v", err)
			}
			if !strings.Contains(string(sent["taxonomyInput"]), `"type":"agent"`) {
				t.Fatalf("input raw: %s", sent["taxonomyInput"])
			}
		})
	}
}

func TestCreate_RequiresName(t *testing.T) {
	c := newTestClient(t, &scriptedTransport{responses: []scriptedResponse{{}}})
	if _, err := c.Create(context.Background(), "", EvaluationTaxonomy{}, nil); err == nil {
		t.Fatal("expected error")
	}
}

func TestUpdate_PatchesBody(t *testing.T) {
	st := &scriptedTransport{responses: []scriptedResponse{{
		body: `{"name":"t1","description":"new"}`,
	}}}
	c := newTestClient(t, st)
	got, err := c.Update(context.Background(), "t1", EvaluationTaxonomy{Description: "new"}, nil)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if got.Description != "new" {
		t.Fatalf("decoded: %+v", got)
	}
	call := st.calls[0]
	if call.method != http.MethodPatch || call.path != "/evaluationtaxonomies/t1" {
		t.Fatalf("method/path: %+v", call)
	}
}

func TestUpdate_RequiresName(t *testing.T) {
	c := newTestClient(t, &scriptedTransport{responses: []scriptedResponse{{}}})
	if _, err := c.Update(context.Background(), "", EvaluationTaxonomy{}, nil); err == nil {
		t.Fatal("expected error")
	}
}

func TestDelete_FiresDelete204(t *testing.T) {
	st := &scriptedTransport{responses: []scriptedResponse{{status: http.StatusNoContent}}}
	c := newTestClient(t, st)
	if _, err := c.Delete(context.Background(), "t1", &DeleteOptions{ClientRequestID: "rd-1"}); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	call := st.calls[0]
	if call.method != http.MethodDelete || call.path != "/evaluationtaxonomies/t1" {
		t.Fatalf("method/path: %+v", call)
	}
	if call.clientReq != "rd-1" {
		t.Fatalf("client-request-id: %q", call.clientReq)
	}
}

func TestDelete_RequiresName(t *testing.T) {
	c := newTestClient(t, &scriptedTransport{responses: []scriptedResponse{{}}})
	if _, err := c.Delete(context.Background(), "", nil); err == nil {
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
