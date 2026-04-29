package connections

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
)

// fakeCred satisfies azcore.TokenCredential without making any network calls.
type fakeCred struct{}

func (fakeCred) GetToken(_ context.Context, _ policy.TokenRequestOptions) (azcore.AccessToken, error) {
	return azcore.AccessToken{Token: "fake", ExpiresOn: time.Now().Add(time.Hour)}, nil
}

// fakeTransport captures the request and returns a canned response.
type fakeTransport struct {
	gotReq *http.Request
	body   string
	status int
}

func (f *fakeTransport) Do(req *http.Request) (*http.Response, error) {
	f.gotReq = req
	status := f.status
	if status == 0 {
		status = http.StatusOK
	}
	return &http.Response{
		Status:     http.StatusText(status),
		StatusCode: status,
		Body:       io.NopCloser(bytes.NewBufferString(f.body)),
		Header:     make(http.Header),
		Request:    req,
	}, nil
}

func newTestClient(t *testing.T, ft *fakeTransport) *Client {
	t.Helper()
	c, err := NewClient("https://example.test", fakeCred{}, &ClientOptions{
		ClientOptions: azcore.ClientOptions{Transport: ft},
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return c
}

func TestNewListPager_FiresGetWithAPIVersion(t *testing.T) {
	ft := &fakeTransport{
		body: `{"value":[{"name":"c1","type":"AzureOpenAI"}]}`,
	}
	c := newTestClient(t, ft)

	pager := c.NewListPager(nil)
	if !pager.More() {
		t.Fatal("pager.More() = false before first fetch, want true")
	}
	page, err := pager.NextPage(context.Background())
	if err != nil {
		t.Fatalf("NextPage: %v", err)
	}

	if got, want := ft.gotReq.Method, http.MethodGet; got != want {
		t.Errorf("method = %s, want %s", got, want)
	}
	if got, want := ft.gotReq.URL.Path, "/connections"; got != want {
		t.Errorf("path = %s, want %s", got, want)
	}
	if got, want := ft.gotReq.URL.Query().Get("api-version"), "v1"; got != want {
		t.Errorf("api-version query = %s, want %s", got, want)
	}

	if len(page.Value) != 1 || page.Value[0].Name != "c1" {
		body, _ := json.Marshal(page)
		t.Errorf("page = %s, want one connection named c1", body)
	}
	if page.Value[0].Type != ConnectionTypeAzureOpenAI {
		t.Errorf("page.Value[0].Type = %q, want %q", page.Value[0].Type, ConnectionTypeAzureOpenAI)
	}
}
