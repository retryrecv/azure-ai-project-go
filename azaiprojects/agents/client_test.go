package agents

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
)

// --- shared test plumbing ---

type fakeCred struct{}

func (fakeCred) GetToken(_ context.Context, _ policy.TokenRequestOptions) (azcore.AccessToken, error) {
	return azcore.AccessToken{Token: "fake", ExpiresOn: time.Now().Add(time.Hour)}, nil
}

// scriptedTransport returns one canned response per call, in order.
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
	method string
	path   string
	query  string
	body   []byte
}

func (s *scriptedTransport) Do(req *http.Request) (*http.Response, error) {
	var body []byte
	if req.Body != nil {
		body, _ = io.ReadAll(req.Body)
	}
	s.calls = append(s.calls, recordedCall{
		method: req.Method,
		path:   req.URL.Path,
		query:  req.URL.RawQuery,
		body:   body,
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

// --- read-op tests ---

func TestNewListPager_FirstPageQueryParams(t *testing.T) {
	st := &scriptedTransport{responses: []scriptedResponse{{
		body: `{"data":[{"object":"agent","id":"a_1","name":"x","versions":{"latest":{"object":"agent.version","id":"av_1","name":"x","version":"1","created_at":1700000000,"definition":{"kind":"prompt"}}}}],"has_more":false}`,
	}}}
	c := newTestClient(t, st)
	limit := int32(10)
	order := PageOrderDesc
	kind := AgentKindPrompt
	pager := c.NewListPager(&ListOptions{Kind: &kind, Limit: &limit, Order: &order})
	page, err := pager.NextPage(context.Background())
	if err != nil {
		t.Fatalf("NextPage: %v", err)
	}
	if len(page.Data) != 1 || page.Data[0].Name != "x" {
		t.Fatalf("decoded page: %+v", page)
	}
	got := st.calls[0]
	if got.method != http.MethodGet || got.path != "/agents" {
		t.Fatalf("method/path: %+v", got)
	}
	for _, want := range []string{"api-version=v1", "kind=prompt", "limit=10", "order=desc"} {
		if !strings.Contains(got.query, want) {
			t.Fatalf("missing %s in query %s", want, got.query)
		}
	}
}

func TestNewListPager_FollowsHasMoreCursor(t *testing.T) {
	st := &scriptedTransport{responses: []scriptedResponse{
		{body: `{"data":[{"object":"agent","id":"a_1","name":"x","versions":{"latest":{"object":"agent.version","id":"av_1","name":"x","version":"1","created_at":1,"definition":{"kind":"prompt"}}}}],"has_more":true,"last_id":"a_1"}`},
		{body: `{"data":[{"object":"agent","id":"a_2","name":"y","versions":{"latest":{"object":"agent.version","id":"av_2","name":"y","version":"1","created_at":1,"definition":{"kind":"prompt"}}}}],"has_more":false}`},
	}}
	c := newTestClient(t, st)
	pager := c.NewListPager(nil)
	var seen []string
	for pager.More() {
		page, err := pager.NextPage(context.Background())
		if err != nil {
			t.Fatalf("NextPage: %v", err)
		}
		for _, a := range page.Data {
			seen = append(seen, a.Name)
		}
	}
	if strings.Join(seen, ",") != "x,y" {
		t.Fatalf("agents seen: %v", seen)
	}
	if len(st.calls) != 2 {
		t.Fatalf("expected 2 calls, got %d", len(st.calls))
	}
	if !strings.Contains(st.calls[1].query, "after=a_1") {
		t.Fatalf("second call missing after cursor: %s", st.calls[1].query)
	}
}

func TestNewListVersionsPager_QueryParams(t *testing.T) {
	st := &scriptedTransport{responses: []scriptedResponse{{
		body: `{"data":[{"object":"agent.version","id":"av_1","name":"x","version":"1","created_at":1700000000,"definition":{"kind":"prompt"}}],"has_more":false}`,
	}}}
	c := newTestClient(t, st)
	limit := int32(5)
	order := PageOrderAsc
	before := "av_99"
	pager := c.NewListVersionsPager("my-agent", &ListVersionsOptions{
		Limit: &limit, Order: &order, Before: &before,
	})
	if _, err := pager.NextPage(context.Background()); err != nil {
		t.Fatalf("NextPage: %v", err)
	}
	got := st.calls[0]
	if got.method != http.MethodGet || got.path != "/agents/my-agent/versions" {
		t.Fatalf("method/path: %+v", got)
	}
	for _, want := range []string{"limit=5", "order=asc", "before=av_99", "api-version=v1"} {
		if !strings.Contains(got.query, want) {
			t.Fatalf("missing %s in query %s", want, got.query)
		}
	}
}

func TestNewListVersionsPager_RequiresName(t *testing.T) {
	st := &scriptedTransport{responses: []scriptedResponse{{body: `{"data":[],"has_more":false}`}}}
	c := newTestClient(t, st)
	pager := c.NewListVersionsPager("", nil)
	if _, err := pager.NextPage(context.Background()); err == nil {
		t.Fatal("expected error for empty agentName")
	}
}

func TestGet_FiresGetWithName(t *testing.T) {
	st := &scriptedTransport{responses: []scriptedResponse{{
		body: `{"object":"agent","id":"a_1","name":"x","versions":{"latest":{"object":"agent.version","id":"av_1","name":"x","version":"1","created_at":1700000000,"definition":{"kind":"hosted","cpu":"1","memory":"2Gi"}}}}`,
	}}}
	c := newTestClient(t, st)
	a, err := c.Get(context.Background(), "my-agent", nil)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if a.Name != "x" || a.Versions.Latest.Version != "1" {
		t.Fatalf("decoded agent: %+v", a)
	}
	got := st.calls[0]
	if got.method != http.MethodGet || got.path != "/agents/my-agent" {
		t.Fatalf("method/path: %+v", got)
	}
	if !strings.Contains(got.query, "api-version=v1") {
		t.Fatalf("query: %s", got.query)
	}
}

func TestGet_RequiresName(t *testing.T) {
	c := newTestClient(t, &scriptedTransport{responses: []scriptedResponse{{}}})
	if _, err := c.Get(context.Background(), "", nil); err == nil {
		t.Fatal("expected error for empty agentName")
	}
}

func TestGetVersion_FiresGetWithBothIDs(t *testing.T) {
	st := &scriptedTransport{responses: []scriptedResponse{{
		body: `{"object":"agent.version","id":"av_1","name":"x","version":"3","created_at":1700000000,"definition":{"kind":"prompt","model":"m"}}`,
	}}}
	c := newTestClient(t, st)
	v, err := c.GetVersion(context.Background(), "my-agent", "3", nil)
	if err != nil {
		t.Fatalf("GetVersion: %v", err)
	}
	if v.Version != "3" {
		t.Fatalf("decoded version: %+v", v)
	}
	got := st.calls[0]
	if got.method != http.MethodGet || got.path != "/agents/my-agent/versions/3" {
		t.Fatalf("method/path: %+v", got)
	}
}

func TestGetVersion_RequiresBoth(t *testing.T) {
	c := newTestClient(t, &scriptedTransport{responses: []scriptedResponse{{}}})
	if _, err := c.GetVersion(context.Background(), "", "1", nil); err == nil {
		t.Fatal("expected error for empty agentName")
	}
	if _, err := c.GetVersion(context.Background(), "x", "", nil); err == nil {
		t.Fatal("expected error for empty agentVersion")
	}
}
