package schedules

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
		body: `{"value":[{"id":"s1","enabled":true,"trigger":{"type":"Cron","expression":"* * * * *"},"task":{"type":"x"}}]}`,
	}}}
	c := newTestClient(t, st)
	ty, en := "Cron", true
	pager := c.NewListPager(&ListOptions{Type: &ty, Enabled: &en})
	page, err := pager.NextPage(context.Background())
	if err != nil {
		t.Fatalf("NextPage: %v", err)
	}
	if len(page.Value) != 1 || page.Value[0].ID != "s1" {
		t.Fatalf("decoded: %+v", page)
	}
	if !strings.Contains(string(page.Value[0].Trigger), `"type":"Cron"`) {
		t.Fatalf("trigger: %s", page.Value[0].Trigger)
	}
	got := st.calls[0]
	if got.method != http.MethodGet || got.path != "/schedules" {
		t.Fatalf("method/path: %+v", got)
	}
	for _, want := range []string{"type=Cron", "enabled=true", "api-version=v1"} {
		if !strings.Contains(got.query, want) {
			t.Fatalf("missing %s in %s", want, got.query)
		}
	}
	if got.foundryFeat != foundryHeader {
		t.Fatalf("foundry: %q", got.foundryFeat)
	}
}

func TestList_FollowsNextLink(t *testing.T) {
	st := &scriptedTransport{responses: []scriptedResponse{
		{body: `{"value":[{"id":"a","enabled":true,"trigger":{},"task":{}}],"nextLink":"https://example.test/schedules?page=2"}`},
		{body: `{"value":[{"id":"b","enabled":false,"trigger":{},"task":{}}]}`},
	}}
	c := newTestClient(t, st)
	pager := c.NewListPager(nil)
	var ids []string
	for pager.More() {
		page, err := pager.NextPage(context.Background())
		if err != nil {
			t.Fatalf("NextPage: %v", err)
		}
		for _, s := range page.Value {
			ids = append(ids, s.ID)
		}
	}
	if strings.Join(ids, ",") != "a,b" {
		t.Fatalf("ids: %v", ids)
	}
	if !strings.Contains(st.calls[1].query, "page=2") {
		t.Fatalf("second call: %s", st.calls[1].query)
	}
}

func TestGet_FiresGetWithID(t *testing.T) {
	st := &scriptedTransport{responses: []scriptedResponse{{
		body: `{"id":"s1","displayName":"Demo","enabled":true,"trigger":{"type":"Cron"},"task":{"type":"foo"}}`,
	}}}
	c := newTestClient(t, st)
	got, err := c.Get(context.Background(), "s1", nil)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ID != "s1" || got.DisplayName != "Demo" {
		t.Fatalf("decoded: %+v", got)
	}
	call := st.calls[0]
	if call.method != http.MethodGet || call.path != "/schedules/s1" {
		t.Fatalf("method/path: %+v", call)
	}
}

func TestGet_RequiresID(t *testing.T) {
	c := newTestClient(t, &scriptedTransport{responses: []scriptedResponse{{}}})
	if _, err := c.Get(context.Background(), "", nil); err == nil {
		t.Fatal("expected error")
	}
}

func TestCreateOrUpdate_PutsBodyAndAcceptsBoth200And201(t *testing.T) {
	for _, status := range []int{http.StatusOK, http.StatusCreated} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			st := &scriptedTransport{responses: []scriptedResponse{{
				status: status,
				body:   `{"id":"s1","enabled":true,"trigger":{},"task":{}}`,
			}}}
			c := newTestClient(t, st)
			sched := Schedule{
				DisplayName: "Demo",
				Enabled:     true,
				Trigger:     json.RawMessage(`{"type":"Cron","expression":"* * * * *"}`),
				Task:        json.RawMessage(`{"type":"foo"}`),
			}
			got, err := c.CreateOrUpdate(context.Background(), "s1", sched, &CreateOrUpdateOptions{ClientRequestID: "req-1"})
			if err != nil {
				t.Fatalf("CreateOrUpdate: %v", err)
			}
			if got.ID != "s1" {
				t.Fatalf("decoded: %+v", got)
			}
			call := st.calls[0]
			if call.method != http.MethodPut || call.path != "/schedules/s1" {
				t.Fatalf("method/path: %+v", call)
			}
			if call.clientReq != "req-1" {
				t.Fatalf("client-request-id: %q", call.clientReq)
			}
			var sent map[string]json.RawMessage
			if err := json.Unmarshal(call.body, &sent); err != nil {
				t.Fatalf("body: %v", err)
			}
			if !strings.Contains(string(sent["trigger"]), `"type":"Cron"`) {
				t.Fatalf("trigger raw: %s", sent["trigger"])
			}
		})
	}
}

func TestCreateOrUpdate_RequiresID(t *testing.T) {
	c := newTestClient(t, &scriptedTransport{responses: []scriptedResponse{{}}})
	if _, err := c.CreateOrUpdate(context.Background(), "", Schedule{}, nil); err == nil {
		t.Fatal("expected error")
	}
}

func TestDelete_FiresDeleteAnd204IsSuccess(t *testing.T) {
	st := &scriptedTransport{responses: []scriptedResponse{{status: http.StatusNoContent}}}
	c := newTestClient(t, st)
	if _, err := c.Delete(context.Background(), "s1", nil); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	call := st.calls[0]
	if call.method != http.MethodDelete || call.path != "/schedules/s1" {
		t.Fatalf("method/path: %+v", call)
	}
}

func TestDelete_RequiresID(t *testing.T) {
	c := newTestClient(t, &scriptedTransport{responses: []scriptedResponse{{}}})
	if _, err := c.Delete(context.Background(), "", nil); err == nil {
		t.Fatal("expected error")
	}
}

func TestListRuns_FiresGetForSchedule(t *testing.T) {
	st := &scriptedTransport{responses: []scriptedResponse{{
		body: `{"value":[{"id":"r1","scheduleId":"s1","success":true}]}`,
	}}}
	c := newTestClient(t, st)
	pager := c.NewListRunsPager("s1", nil)
	page, err := pager.NextPage(context.Background())
	if err != nil {
		t.Fatalf("NextPage: %v", err)
	}
	if len(page.Value) != 1 || page.Value[0].ID != "r1" {
		t.Fatalf("decoded: %+v", page)
	}
	call := st.calls[0]
	if call.path != "/schedules/s1/runs" {
		t.Fatalf("path: %s", call.path)
	}
}

func TestListRuns_RequiresID(t *testing.T) {
	c := newTestClient(t, &scriptedTransport{responses: []scriptedResponse{{}}})
	pager := c.NewListRunsPager("", nil)
	if _, err := pager.NextPage(context.Background()); err == nil {
		t.Fatal("expected error")
	}
}

func TestGetRun_FiresGet(t *testing.T) {
	st := &scriptedTransport{responses: []scriptedResponse{{
		body: `{"id":"r1","scheduleId":"s1","success":true}`,
	}}}
	c := newTestClient(t, st)
	got, err := c.GetRun(context.Background(), "s1", "r1", nil)
	if err != nil {
		t.Fatalf("GetRun: %v", err)
	}
	if got.ID != "r1" || !got.Success {
		t.Fatalf("decoded: %+v", got)
	}
	call := st.calls[0]
	if call.path != "/schedules/s1/runs/r1" {
		t.Fatalf("path: %s", call.path)
	}
}

func TestGetRun_Requires(t *testing.T) {
	c := newTestClient(t, &scriptedTransport{responses: []scriptedResponse{{}}})
	if _, err := c.GetRun(context.Background(), "", "r", nil); err == nil {
		t.Fatal("expected error for empty scheduleID")
	}
	if _, err := c.GetRun(context.Background(), "s", "", nil); err == nil {
		t.Fatal("expected error for empty runID")
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
