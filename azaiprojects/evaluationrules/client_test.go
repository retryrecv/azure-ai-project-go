package evaluationrules

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

func TestNewListPager_FiresGetWithFilters(t *testing.T) {
	st := &scriptedTransport{responses: []scriptedResponse{{
		body: `{"value":[{"id":"r1","action":{"type":"continuousEvaluation","evalId":"e1"},"eventType":"manual","enabled":true}]}`,
	}}}
	c := newTestClient(t, st)
	at := EvaluationRuleActionTypeContinuousEvaluation
	an := "my-agent"
	en := true
	pager := c.NewListPager(&ListOptions{ActionType: &at, AgentName: &an, Enabled: &en})
	page, err := pager.NextPage(context.Background())
	if err != nil {
		t.Fatalf("NextPage: %v", err)
	}
	if len(page.Value) != 1 || page.Value[0].ID != "r1" {
		t.Fatalf("decoded page: %+v", page)
	}
	got := st.calls[0]
	if got.method != http.MethodGet || got.path != "/evaluationrules" {
		t.Fatalf("method/path: %+v", got)
	}
	for _, want := range []string{"actionType=continuousEvaluation", "agentName=my-agent", "enabled=true", "api-version=v1"} {
		if !strings.Contains(got.query, want) {
			t.Fatalf("missing %s in query %s", want, got.query)
		}
	}
}

func TestNewListPager_FollowsNextLink(t *testing.T) {
	st := &scriptedTransport{responses: []scriptedResponse{
		{body: `{"value":[{"id":"r1","action":{"type":"continuousEvaluation","evalId":"e1"},"eventType":"manual","enabled":true}],"nextLink":"https://example.test/evaluationrules?page=2"}`},
		{body: `{"value":[{"id":"r2","action":{"type":"humanEvaluationPreview","templateId":"t1"},"eventType":"manual","enabled":true}]}`},
	}}
	c := newTestClient(t, st)
	pager := c.NewListPager(nil)
	var seen []string
	for pager.More() {
		page, err := pager.NextPage(context.Background())
		if err != nil {
			t.Fatalf("NextPage: %v", err)
		}
		for _, r := range page.Value {
			seen = append(seen, r.ID)
		}
	}
	if strings.Join(seen, ",") != "r1,r2" {
		t.Fatalf("rules seen: %v", seen)
	}
	if len(st.calls) != 2 {
		t.Fatalf("expected 2 calls, got %d", len(st.calls))
	}
	if !strings.Contains(st.calls[1].query, "page=2") {
		t.Fatalf("second call should follow nextLink: %s", st.calls[1].query)
	}
}

func TestGet_FiresGetWithID(t *testing.T) {
	st := &scriptedTransport{responses: []scriptedResponse{{
		body: `{"id":"r1","displayName":"Demo","action":{"type":"continuousEvaluation","evalId":"e1","maxHourlyRuns":2},"eventType":"responseCompleted","enabled":true}`,
	}}}
	c := newTestClient(t, st)
	got, err := c.Get(context.Background(), "r1", nil)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ID != "r1" || got.DisplayName != "Demo" {
		t.Fatalf("decoded: %+v", got)
	}
	c2, ok := got.Action.Value.(ContinuousEvaluationRuleAction)
	if !ok {
		t.Fatalf("action type: %T", got.Action.Value)
	}
	if c2.EvalID != "e1" || c2.MaxHourlyRuns == nil || *c2.MaxHourlyRuns != 2 {
		t.Fatalf("action contents: %+v", c2)
	}
	call := st.calls[0]
	if call.method != http.MethodGet || call.path != "/evaluationrules/r1" {
		t.Fatalf("method/path: %+v", call)
	}
	if !strings.Contains(call.query, "api-version=v1") {
		t.Fatalf("query: %s", call.query)
	}
}

func TestGet_RequiresID(t *testing.T) {
	c := newTestClient(t, &scriptedTransport{responses: []scriptedResponse{{}}})
	if _, err := c.Get(context.Background(), "", nil); err == nil {
		t.Fatal("expected error for empty id")
	}
}

func TestCreateOrUpdate_PutsBodyAndAcceptsBoth200And201(t *testing.T) {
	for _, status := range []int{http.StatusOK, http.StatusCreated} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			st := &scriptedTransport{responses: []scriptedResponse{{
				status: status,
				body:   `{"id":"r1","action":{"type":"humanEvaluationPreview","templateId":"t1"},"eventType":"manual","enabled":true}`,
			}}}
			c := newTestClient(t, st)
			rule := EvaluationRule{
				DisplayName: "Demo",
				EventType:   EvaluationRuleEventTypeManual,
				Enabled:     true,
				Action: EvaluationRuleActionValue{
					Value: HumanEvaluationPreviewRuleAction{
						Type:       EvaluationRuleActionTypeHumanEvaluationPreview,
						TemplateID: "t1",
					},
				},
			}
			got, err := c.CreateOrUpdate(context.Background(), "r1", rule, nil)
			if err != nil {
				t.Fatalf("CreateOrUpdate: %v", err)
			}
			if got.ID != "r1" {
				t.Fatalf("decoded: %+v", got)
			}
			call := st.calls[0]
			if call.method != http.MethodPut || call.path != "/evaluationrules/r1" {
				t.Fatalf("method/path: %+v", call)
			}
			var sent struct {
				DisplayName string          `json:"displayName"`
				EventType   string          `json:"eventType"`
				Enabled     bool            `json:"enabled"`
				Action      json.RawMessage `json:"action"`
			}
			if err := json.Unmarshal(call.body, &sent); err != nil {
				t.Fatalf("body unmarshal: %v", err)
			}
			if sent.DisplayName != "Demo" || sent.EventType != "manual" || !sent.Enabled {
				t.Fatalf("body fields: %+v", sent)
			}
			if !strings.Contains(string(sent.Action), `"type":"humanEvaluationPreview"`) ||
				!strings.Contains(string(sent.Action), `"templateId":"t1"`) {
				t.Fatalf("action JSON missing fields: %s", sent.Action)
			}
		})
	}
}

func TestCreateOrUpdate_RequiresID(t *testing.T) {
	c := newTestClient(t, &scriptedTransport{responses: []scriptedResponse{{}}})
	if _, err := c.CreateOrUpdate(context.Background(), "", EvaluationRule{}, nil); err == nil {
		t.Fatal("expected error for empty id")
	}
}

func TestDelete_FiresDeleteAnd204IsSuccess(t *testing.T) {
	st := &scriptedTransport{responses: []scriptedResponse{{status: http.StatusNoContent}}}
	c := newTestClient(t, st)
	if _, err := c.Delete(context.Background(), "r1", nil); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	call := st.calls[0]
	if call.method != http.MethodDelete || call.path != "/evaluationrules/r1" {
		t.Fatalf("method/path: %+v", call)
	}
}

func TestDelete_RequiresID(t *testing.T) {
	c := newTestClient(t, &scriptedTransport{responses: []scriptedResponse{{}}})
	if _, err := c.Delete(context.Background(), "", nil); err == nil {
		t.Fatal("expected error for empty id")
	}
}
