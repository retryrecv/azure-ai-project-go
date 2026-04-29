package skills

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
	status      int
	body        string
	contentType string
}

type recordedCall struct {
	method      string
	path        string
	query       string
	body        []byte
	contentType string
	foundryFeat string
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
	hdr := make(http.Header)
	if resp.contentType != "" {
		hdr.Set("Content-Type", resp.contentType)
	}
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(bytes.NewBufferString(resp.body)),
		Header:     hdr,
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
		body: `{"data":[{"skill_id":"s1","has_blob":false,"name":"sk1"}],"last_id":"s1","has_more":false}`,
	}}}
	c := newTestClient(t, st)
	limit := int32(25)
	order := PageOrderDesc
	pager := c.NewListPager(&ListOptions{Limit: &limit, Order: &order})
	page, err := pager.NextPage(context.Background())
	if err != nil {
		t.Fatalf("NextPage: %v", err)
	}
	if len(page.Data) != 1 || page.Data[0].Name != "sk1" {
		t.Fatalf("decoded: %+v", page)
	}
	got := st.calls[0]
	if got.method != http.MethodGet || got.path != "/skills" {
		t.Fatalf("method/path: %+v", got)
	}
	for _, want := range []string{"limit=25", "order=desc", "api-version=v1"} {
		if !strings.Contains(got.query, want) {
			t.Fatalf("missing %s in query %s", want, got.query)
		}
	}
	if got.foundryFeat != foundryHeader {
		t.Fatalf("foundry-features: %q", got.foundryFeat)
	}
}

func TestList_PagesViaLastID(t *testing.T) {
	st := &scriptedTransport{responses: []scriptedResponse{
		{body: `{"data":[{"skill_id":"a","has_blob":false,"name":"a"}],"last_id":"a","has_more":true}`},
		{body: `{"data":[{"skill_id":"b","has_blob":false,"name":"b"}],"last_id":"b","has_more":false}`},
	}}
	c := newTestClient(t, st)
	pager := c.NewListPager(nil)
	var seen []string
	for pager.More() {
		page, err := pager.NextPage(context.Background())
		if err != nil {
			t.Fatalf("NextPage: %v", err)
		}
		for _, s := range page.Data {
			seen = append(seen, s.Name)
		}
	}
	if strings.Join(seen, ",") != "a,b" {
		t.Fatalf("names: %v", seen)
	}
	if !strings.Contains(st.calls[1].query, "after=a") {
		t.Fatalf("second call should set after=a: %s", st.calls[1].query)
	}
}

func TestGet_FiresGetWithName(t *testing.T) {
	st := &scriptedTransport{responses: []scriptedResponse{{
		body: `{"skill_id":"s1","has_blob":true,"name":"sk1","description":"d","metadata":{"k":"v"}}`,
	}}}
	c := newTestClient(t, st)
	got, err := c.Get(context.Background(), "sk1", nil)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Name != "sk1" || got.SkillID != "s1" || !got.HasBlob || got.Metadata["k"] != "v" {
		t.Fatalf("decoded: %+v", got)
	}
	call := st.calls[0]
	if call.method != http.MethodGet || call.path != "/skills/sk1" {
		t.Fatalf("method/path: %+v", call)
	}
	if call.foundryFeat != foundryHeader {
		t.Fatalf("foundry-features: %q", call.foundryFeat)
	}
}

func TestGet_RequiresName(t *testing.T) {
	c := newTestClient(t, &scriptedTransport{responses: []scriptedResponse{{}}})
	if _, err := c.Get(context.Background(), "", nil); err == nil {
		t.Fatal("expected error")
	}
}

func TestCreate_PostsBodyAndExpects201(t *testing.T) {
	st := &scriptedTransport{responses: []scriptedResponse{{
		status: http.StatusCreated,
		body:   `{"skill_id":"s1","has_blob":false,"name":"sk1"}`,
	}}}
	c := newTestClient(t, st)
	got, err := c.Create(context.Background(), CreateSkillBody{
		Name: "sk1", Description: "d", Instructions: "i",
	}, nil)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if got.Name != "sk1" {
		t.Fatalf("decoded: %+v", got)
	}
	call := st.calls[0]
	if call.method != http.MethodPost || call.path != "/skills" {
		t.Fatalf("method/path: %+v", call)
	}
	if !strings.HasPrefix(call.contentType, "application/json") {
		t.Fatalf("content-type: %q", call.contentType)
	}
	var sent map[string]any
	if err := json.Unmarshal(call.body, &sent); err != nil {
		t.Fatalf("body decode: %v", err)
	}
	if sent["name"] != "sk1" || sent["description"] != "d" || sent["instructions"] != "i" {
		t.Fatalf("body fields: %v", sent)
	}
}

func TestCreate_RequiresName(t *testing.T) {
	c := newTestClient(t, &scriptedTransport{responses: []scriptedResponse{{}}})
	if _, err := c.Create(context.Background(), CreateSkillBody{}, nil); err == nil {
		t.Fatal("expected error")
	}
}

func TestUpdate_PostsBodyTo200(t *testing.T) {
	st := &scriptedTransport{responses: []scriptedResponse{{
		body: `{"skill_id":"s1","has_blob":false,"name":"sk1","description":"d2"}`,
	}}}
	c := newTestClient(t, st)
	got, err := c.Update(context.Background(), "sk1", UpdateSkillBody{Description: "d2"}, nil)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if got.Description != "d2" {
		t.Fatalf("decoded: %+v", got)
	}
	call := st.calls[0]
	if call.method != http.MethodPost || call.path != "/skills/sk1" {
		t.Fatalf("method/path: %+v", call)
	}
	var sent map[string]any
	if err := json.Unmarshal(call.body, &sent); err != nil {
		t.Fatalf("body decode: %v", err)
	}
	if sent["description"] != "d2" {
		t.Fatalf("body fields: %v", sent)
	}
}

func TestUpdate_RequiresName(t *testing.T) {
	c := newTestClient(t, &scriptedTransport{responses: []scriptedResponse{{}}})
	if _, err := c.Update(context.Background(), "", UpdateSkillBody{}, nil); err == nil {
		t.Fatal("expected error")
	}
}

func TestDelete_FiresDelete(t *testing.T) {
	st := &scriptedTransport{responses: []scriptedResponse{{
		body: `{"name":"sk1","deleted":true}`,
	}}}
	c := newTestClient(t, st)
	got, err := c.Delete(context.Background(), "sk1", nil)
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if !got.Deleted || got.Name != "sk1" {
		t.Fatalf("decoded: %+v", got)
	}
	call := st.calls[0]
	if call.method != http.MethodDelete || call.path != "/skills/sk1" {
		t.Fatalf("method/path: %+v", call)
	}
}

func TestDelete_RequiresName(t *testing.T) {
	c := newTestClient(t, &scriptedTransport{responses: []scriptedResponse{{}}})
	if _, err := c.Delete(context.Background(), "", nil); err == nil {
		t.Fatal("expected error")
	}
}

func TestDownload_StreamsBinary(t *testing.T) {
	st := &scriptedTransport{responses: []scriptedResponse{{
		body:        "PKZIPDATA",
		contentType: "application/zip",
	}}}
	c := newTestClient(t, st)
	got, err := c.Download(context.Background(), "sk1", nil)
	if err != nil {
		t.Fatalf("Download: %v", err)
	}
	defer got.Body.Close()
	data, _ := io.ReadAll(got.Body)
	if string(data) != "PKZIPDATA" {
		t.Fatalf("body: %q", data)
	}
	call := st.calls[0]
	if call.method != http.MethodGet || call.path != "/skills/sk1:download" {
		t.Fatalf("method/path: %+v", call)
	}
	if call.accept != "application/zip" {
		t.Fatalf("accept: %q", call.accept)
	}
}

func TestDownload_RequiresName(t *testing.T) {
	c := newTestClient(t, &scriptedTransport{responses: []scriptedResponse{{}}})
	if _, err := c.Download(context.Background(), "", nil); err == nil {
		t.Fatal("expected error")
	}
}

func TestCreateFromPackage_PostsZipExpects201(t *testing.T) {
	st := &scriptedTransport{responses: []scriptedResponse{{
		status: http.StatusCreated,
		body:   `{"skill_id":"s1","has_blob":true,"name":"sk1"}`,
	}}}
	c := newTestClient(t, st)
	zipData := []byte("PKZIPCONTENT")
	got, err := c.CreateFromPackage(context.Background(), zipData, nil)
	if err != nil {
		t.Fatalf("CreateFromPackage: %v", err)
	}
	if !got.HasBlob {
		t.Fatalf("decoded: %+v", got)
	}
	call := st.calls[0]
	if call.method != http.MethodPost || call.path != "/skills:import" {
		t.Fatalf("method/path: %+v", call)
	}
	if !strings.HasPrefix(call.contentType, "application/zip") {
		t.Fatalf("content-type: %q", call.contentType)
	}
	if !bytes.Equal(call.body, zipData) {
		t.Fatalf("body: %q", call.body)
	}
}

func TestCreateFromPackage_RequiresPkg(t *testing.T) {
	c := newTestClient(t, &scriptedTransport{responses: []scriptedResponse{{}}})
	if _, err := c.CreateFromPackage(context.Background(), nil, nil); err == nil {
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

func TestNewClientFromPipeline_DefaultsAPIVersion(t *testing.T) {
	parent, _ := NewClient("https://example.test", fakeCred{}, nil)
	c := NewClientFromPipeline("https://example.test", "", parent.pl)
	if c.apiVersion != defaultAPIVer {
		t.Fatalf("apiVersion: %q", c.apiVersion)
	}
}
