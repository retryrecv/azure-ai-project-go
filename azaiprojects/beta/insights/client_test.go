package insights

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
	repeatID    string
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
		repeatID:    req.Header.Get("repeatability-request-id"),
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
		body: `{"value":[{"id":"i1","displayName":"d","state":"Succeeded"}]}`,
	}}}
	c := newTestClient(t, st)
	ty := "EvaluationRunClusterInsight"
	include := true
	pager := c.NewListPager(&ListOptions{InsightType: &ty, IncludeCoordinates: &include, ClientRequestID: "req-1"})
	page, err := pager.NextPage(context.Background())
	if err != nil {
		t.Fatalf("NextPage: %v", err)
	}
	if len(page.Value) != 1 || page.Value[0].InsightID != "i1" {
		t.Fatalf("decoded: %+v", page)
	}
	call := st.calls[0]
	if call.method != http.MethodGet || call.path != "/insights" {
		t.Fatalf("method/path: %+v", call)
	}
	for _, want := range []string{"type=EvaluationRunClusterInsight", "includeCoordinates=true", "api-version=v1"} {
		if !strings.Contains(call.query, want) {
			t.Fatalf("missing %s in %s", want, call.query)
		}
	}
	if call.clientReq != "req-1" {
		t.Fatalf("client-request-id: %q", call.clientReq)
	}
	if call.foundryFeat != foundryHeader {
		t.Fatalf("foundry: %q", call.foundryFeat)
	}
}

func TestList_FollowsNextLink(t *testing.T) {
	st := &scriptedTransport{responses: []scriptedResponse{
		{body: `{"value":[{"id":"a"}],"nextLink":"https://example.test/insights?page=2"}`},
		{body: `{"value":[{"id":"b"}]}`},
	}}
	c := newTestClient(t, st)
	pager := c.NewListPager(nil)
	var ids []string
	for pager.More() {
		page, err := pager.NextPage(context.Background())
		if err != nil {
			t.Fatalf("NextPage: %v", err)
		}
		for _, in := range page.Value {
			ids = append(ids, in.InsightID)
		}
	}
	if strings.Join(ids, ",") != "a,b" {
		t.Fatalf("ids: %v", ids)
	}
	if !strings.Contains(st.calls[1].query, "page=2") {
		t.Fatalf("second call: %s", st.calls[1].query)
	}
}

func TestGet_FiresGetWithIncludeCoords(t *testing.T) {
	st := &scriptedTransport{responses: []scriptedResponse{{
		body: `{"id":"i1","displayName":"d"}`,
	}}}
	c := newTestClient(t, st)
	include := true
	got, err := c.Get(context.Background(), "i1", &GetOptions{IncludeCoordinates: &include})
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.InsightID != "i1" {
		t.Fatalf("decoded: %+v", got)
	}
	call := st.calls[0]
	if call.method != http.MethodGet || call.path != "/insights/i1" {
		t.Fatalf("method/path: %+v", call)
	}
	if !strings.Contains(call.query, "includeCoordinates=true") {
		t.Fatalf("query: %s", call.query)
	}
}

func TestGet_RequiresID(t *testing.T) {
	c := newTestClient(t, &scriptedTransport{responses: []scriptedResponse{{}}})
	if _, err := c.Get(context.Background(), "", nil); err == nil {
		t.Fatal("expected error")
	}
}

func TestGenerate_PostsBodyExpects201(t *testing.T) {
	st := &scriptedTransport{responses: []scriptedResponse{{
		status: http.StatusCreated,
		body:   `{"id":"i1","displayName":"d","state":"Running"}`,
	}}}
	c := newTestClient(t, st)
	got, err := c.Generate(context.Background(), Insight{
		DisplayName: "d",
		Request:     json.RawMessage(`{"type":"EvaluationRunClusterInsight","evalId":"e1"}`),
	}, &GenerateOptions{RepeatabilityRequestID: "rep-1"})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if got.InsightID != "i1" {
		t.Fatalf("decoded: %+v", got)
	}
	call := st.calls[0]
	if call.method != http.MethodPost || call.path != "/insights" {
		t.Fatalf("method/path: %+v", call)
	}
	if call.repeatID != "rep-1" {
		t.Fatalf("repeatability-request-id: %q", call.repeatID)
	}
	var sent map[string]json.RawMessage
	if err := json.Unmarshal(call.body, &sent); err != nil {
		t.Fatalf("body: %v", err)
	}
	if !strings.Contains(string(sent["request"]), `"type":"EvaluationRunClusterInsight"`) {
		t.Fatalf("request raw: %s", sent["request"])
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
