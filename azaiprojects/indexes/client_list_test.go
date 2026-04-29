package indexes

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
)

type fakeCred struct{}

func (fakeCred) GetToken(_ context.Context, _ policy.TokenRequestOptions) (azcore.AccessToken, error) {
	return azcore.AccessToken{Token: "fake", ExpiresOn: time.Now().Add(time.Hour)}, nil
}

type fakeTransport struct {
	gotReq    *http.Request
	gotBody   []byte
	body      string
	status    int
	calls     int
	allCalls  []*http.Request
	allBodies [][]byte
}

func (f *fakeTransport) Do(req *http.Request) (*http.Response, error) {
	f.gotReq = req
	f.calls++
	f.allCalls = append(f.allCalls, req)
	if req.Body != nil {
		body, _ := io.ReadAll(req.Body)
		f.gotBody = body
		f.allBodies = append(f.allBodies, body)
	} else {
		f.allBodies = append(f.allBodies, nil)
	}
	status := f.status
	if status == 0 {
		status = http.StatusOK
	}
	return &http.Response{
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

func TestNewListPager_FiresGet(t *testing.T) {
	ft := &fakeTransport{body: `{"value":[{"name":"my-azure-search-index","version":"1.0","type":"AzureSearch"}]}`}
	c := newTestClient(t, ft)

	page, err := c.NewListPager(nil).NextPage(context.Background())
	if err != nil {
		t.Fatalf("NextPage: %v", err)
	}
	if got := ft.gotReq.URL.Path; got != "/indexes" {
		t.Errorf("path = %s, want /indexes", got)
	}
	if got := ft.gotReq.URL.Query().Get("api-version"); got != "v1" {
		t.Errorf("api-version = %s, want v1", got)
	}
	if len(page.Value) != 1 || page.Value[0].Name != "my-azure-search-index" {
		t.Errorf("page = %+v, want one index", page)
	}
}
