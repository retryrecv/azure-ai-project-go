package memorystores

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

func TestList_FiresGetWithCursor(t *testing.T) {
	st := &scriptedTransport{responses: []scriptedResponse{{
		body: `{"data":[{"name":"m1","object":"memory_store","created_at":1700000000,"updated_at":1700000000}],"has_more":false}`,
	}}}
	c := newTestClient(t, st)
	limit := int32(25)
	order := PageOrderDesc
	pager := c.NewListPager(&ListOptions{Limit: &limit, Order: &order})
	page, err := pager.NextPage(context.Background())
	if err != nil {
		t.Fatalf("NextPage: %v", err)
	}
	if len(page.Data) != 1 || page.Data[0].Name != "m1" {
		t.Fatalf("decoded: %+v", page)
	}
	call := st.calls[0]
	if call.method != http.MethodGet || call.path != "/memory_stores" {
		t.Fatalf("method/path: %+v", call)
	}
	for _, want := range []string{"limit=25", "order=desc", "api-version=v1"} {
		if !strings.Contains(call.query, want) {
			t.Fatalf("missing %s in %s", want, call.query)
		}
	}
	if call.foundryFeat != foundryHeader {
		t.Fatalf("foundry: %q", call.foundryFeat)
	}
}

func TestList_FollowsCursor(t *testing.T) {
	st := &scriptedTransport{responses: []scriptedResponse{
		{body: `{"data":[{"name":"a"}],"last_id":"a","has_more":true}`},
		{body: `{"data":[{"name":"b"}],"has_more":false}`},
	}}
	c := newTestClient(t, st)
	pager := c.NewListPager(nil)
	var names []string
	for pager.More() {
		page, err := pager.NextPage(context.Background())
		if err != nil {
			t.Fatalf("NextPage: %v", err)
		}
		for _, m := range page.Data {
			names = append(names, m.Name)
		}
	}
	if strings.Join(names, ",") != "a,b" {
		t.Fatalf("names: %v", names)
	}
	if !strings.Contains(st.calls[1].query, "after=a") {
		t.Fatalf("second call: %s", st.calls[1].query)
	}
}

func TestGet_FiresGet(t *testing.T) {
	st := &scriptedTransport{responses: []scriptedResponse{{
		body: `{"name":"m1","object":"memory_store","description":"d"}`,
	}}}
	c := newTestClient(t, st)
	got, err := c.Get(context.Background(), "m1", nil)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Name != "m1" || got.Description != "d" {
		t.Fatalf("decoded: %+v", got)
	}
	call := st.calls[0]
	if call.method != http.MethodGet || call.path != "/memory_stores/m1" {
		t.Fatalf("method/path: %+v", call)
	}
}

func TestGet_RequiresName(t *testing.T) {
	c := newTestClient(t, &scriptedTransport{responses: []scriptedResponse{{}}})
	if _, err := c.Get(context.Background(), "", nil); err == nil {
		t.Fatal("expected error")
	}
}

func TestCreate_PostsBody(t *testing.T) {
	st := &scriptedTransport{responses: []scriptedResponse{{
		body: `{"name":"m1","object":"memory_store"}`,
	}}}
	c := newTestClient(t, st)
	got, err := c.Create(context.Background(), CreateMemoryStoreBody{
		Name:       "m1",
		Definition: json.RawMessage(`{"type":"default"}`),
	}, nil)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if got.Name != "m1" {
		t.Fatalf("decoded: %+v", got)
	}
	call := st.calls[0]
	if call.method != http.MethodPost || call.path != "/memory_stores" {
		t.Fatalf("method/path: %+v", call)
	}
	var sent map[string]json.RawMessage
	if err := json.Unmarshal(call.body, &sent); err != nil {
		t.Fatalf("body: %v", err)
	}
	if !strings.Contains(string(sent["definition"]), `"type":"default"`) {
		t.Fatalf("definition: %s", sent["definition"])
	}
}

func TestCreate_RequiresName(t *testing.T) {
	c := newTestClient(t, &scriptedTransport{responses: []scriptedResponse{{}}})
	if _, err := c.Create(context.Background(), CreateMemoryStoreBody{Definition: json.RawMessage(`{}`)}, nil); err == nil {
		t.Fatal("expected error")
	}
}

func TestCreate_RequiresDefinition(t *testing.T) {
	c := newTestClient(t, &scriptedTransport{responses: []scriptedResponse{{}}})
	if _, err := c.Create(context.Background(), CreateMemoryStoreBody{Name: "x"}, nil); err == nil {
		t.Fatal("expected error")
	}
}

func TestUpdate_PostsBody(t *testing.T) {
	st := &scriptedTransport{responses: []scriptedResponse{{
		body: `{"name":"m1","description":"new"}`,
	}}}
	c := newTestClient(t, st)
	got, err := c.Update(context.Background(), "m1", UpdateMemoryStoreBody{Description: "new"}, nil)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if got.Description != "new" {
		t.Fatalf("decoded: %+v", got)
	}
	call := st.calls[0]
	if call.method != http.MethodPost || call.path != "/memory_stores/m1" {
		t.Fatalf("method/path: %+v", call)
	}
}

func TestUpdate_RequiresName(t *testing.T) {
	c := newTestClient(t, &scriptedTransport{responses: []scriptedResponse{{}}})
	if _, err := c.Update(context.Background(), "", UpdateMemoryStoreBody{}, nil); err == nil {
		t.Fatal("expected error")
	}
}

func TestDelete_FiresDelete(t *testing.T) {
	st := &scriptedTransport{responses: []scriptedResponse{{
		body: `{"object":"memory_store.deleted","name":"m1","deleted":true}`,
	}}}
	c := newTestClient(t, st)
	got, err := c.Delete(context.Background(), "m1", nil)
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if !got.Deleted {
		t.Fatalf("decoded: %+v", got)
	}
	call := st.calls[0]
	if call.method != http.MethodDelete || call.path != "/memory_stores/m1" {
		t.Fatalf("method/path: %+v", call)
	}
}

func TestDelete_RequiresName(t *testing.T) {
	c := newTestClient(t, &scriptedTransport{responses: []scriptedResponse{{}}})
	if _, err := c.Delete(context.Background(), "", nil); err == nil {
		t.Fatal("expected error")
	}
}

func TestDeleteScope_PostsScope(t *testing.T) {
	st := &scriptedTransport{responses: []scriptedResponse{{
		body: `{"object":"memory_store.scope.deleted","name":"m1","scope":"user-1","deleted":true}`,
	}}}
	c := newTestClient(t, st)
	got, err := c.DeleteScope(context.Background(), "m1", "user-1", nil)
	if err != nil {
		t.Fatalf("DeleteScope: %v", err)
	}
	if got.Scope != "user-1" || !got.Deleted {
		t.Fatalf("decoded: %+v", got)
	}
	call := st.calls[0]
	if call.method != http.MethodPost || call.path != "/memory_stores/m1:delete_scope" {
		t.Fatalf("method/path: %+v", call)
	}
	var sent map[string]string
	if err := json.Unmarshal(call.body, &sent); err != nil {
		t.Fatalf("body: %v", err)
	}
	if sent["scope"] != "user-1" {
		t.Fatalf("body: %v", sent)
	}
}

func TestDeleteScope_Requires(t *testing.T) {
	c := newTestClient(t, &scriptedTransport{responses: []scriptedResponse{{}}})
	if _, err := c.DeleteScope(context.Background(), "", "s", nil); err == nil {
		t.Fatal("expected error for empty name")
	}
	if _, err := c.DeleteScope(context.Background(), "m", "", nil); err == nil {
		t.Fatal("expected error for empty scope")
	}
}

func TestUpdateMemories_AcceptsMultipleStatuses(t *testing.T) {
	for _, status := range []int{http.StatusOK, http.StatusCreated, http.StatusAccepted} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			st := &scriptedTransport{responses: []scriptedResponse{{
				status: status,
				body:   `{"update_id":"u1","status":"queued"}`,
			}}}
			c := newTestClient(t, st)
			delay := int32(0)
			body := UpdateMemoriesBody{
				Scope:             "user-1",
				Items:             []json.RawMessage{json.RawMessage(`{"role":"user","content":"hi","type":"message"}`)},
				UpdateDelayInSecs: &delay,
			}
			got, err := c.UpdateMemories(context.Background(), "m1", body, nil)
			if err != nil {
				t.Fatalf("UpdateMemories: %v", err)
			}
			if got.UpdateID != "u1" || got.Status != "queued" {
				t.Fatalf("decoded: %+v", got)
			}
			call := st.calls[0]
			if call.method != http.MethodPost || call.path != "/memory_stores/m1:update_memories" {
				t.Fatalf("method/path: %+v", call)
			}
			var sent map[string]json.RawMessage
			if err := json.Unmarshal(call.body, &sent); err != nil {
				t.Fatalf("body: %v", err)
			}
			if string(sent["scope"]) != `"user-1"` {
				t.Fatalf("scope: %s", sent["scope"])
			}
			if string(sent["update_delay"]) != "0" {
				t.Fatalf("update_delay: %s", sent["update_delay"])
			}
		})
	}
}

func TestUpdateMemories_Requires(t *testing.T) {
	c := newTestClient(t, &scriptedTransport{responses: []scriptedResponse{{}}})
	if _, err := c.UpdateMemories(context.Background(), "", UpdateMemoriesBody{Scope: "s"}, nil); err == nil {
		t.Fatal("expected error for empty name")
	}
	if _, err := c.UpdateMemories(context.Background(), "m", UpdateMemoriesBody{}, nil); err == nil {
		t.Fatal("expected error for empty scope")
	}
}

func TestGetUpdateResult_FiresGet(t *testing.T) {
	st := &scriptedTransport{responses: []scriptedResponse{{
		body: `{"update_id":"u1","status":"completed","result":{"memory_operations":[],"usage":{}}}`,
	}}}
	c := newTestClient(t, st)
	got, err := c.GetUpdateResult(context.Background(), "m1", "u1", nil)
	if err != nil {
		t.Fatalf("GetUpdateResult: %v", err)
	}
	if got.Status != "completed" || len(got.Result) == 0 {
		t.Fatalf("decoded: %+v", got)
	}
	call := st.calls[0]
	if call.method != http.MethodGet || call.path != "/memory_stores/m1/updates/u1" {
		t.Fatalf("method/path: %+v", call)
	}
}

func TestGetUpdateResult_Requires(t *testing.T) {
	c := newTestClient(t, &scriptedTransport{responses: []scriptedResponse{{}}})
	if _, err := c.GetUpdateResult(context.Background(), "", "u", nil); err == nil {
		t.Fatal("expected error for empty name")
	}
	if _, err := c.GetUpdateResult(context.Background(), "m", "", nil); err == nil {
		t.Fatal("expected error for empty updateID")
	}
}

func TestSearchMemories_PostsBody(t *testing.T) {
	st := &scriptedTransport{responses: []scriptedResponse{{
		body: `{"search_id":"s1","memories":[],"usage":{}}`,
	}}}
	c := newTestClient(t, st)
	got, err := c.SearchMemories(context.Background(), "m1", SearchMemoriesBody{
		Scope:   "user-1",
		Items:   []json.RawMessage{json.RawMessage(`{"role":"user","content":"q","type":"message"}`)},
		Options: json.RawMessage(`{"top_k":5}`),
	}, nil)
	if err != nil {
		t.Fatalf("SearchMemories: %v", err)
	}
	if got.SearchID != "s1" {
		t.Fatalf("decoded: %+v", got)
	}
	call := st.calls[0]
	if call.method != http.MethodPost || call.path != "/memory_stores/m1:search_memories" {
		t.Fatalf("method/path: %+v", call)
	}
	var sent map[string]json.RawMessage
	if err := json.Unmarshal(call.body, &sent); err != nil {
		t.Fatalf("body: %v", err)
	}
	if !strings.Contains(string(sent["options"]), `"top_k":5`) {
		t.Fatalf("options: %s", sent["options"])
	}
}

func TestSearchMemories_Requires(t *testing.T) {
	c := newTestClient(t, &scriptedTransport{responses: []scriptedResponse{{}}})
	if _, err := c.SearchMemories(context.Background(), "", SearchMemoriesBody{Scope: "s"}, nil); err == nil {
		t.Fatal("expected error for empty name")
	}
	if _, err := c.SearchMemories(context.Background(), "m", SearchMemoriesBody{}, nil); err == nil {
		t.Fatal("expected error for empty scope")
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
