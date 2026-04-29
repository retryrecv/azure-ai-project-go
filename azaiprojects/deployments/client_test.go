package deployments

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

func ptr[T any](v T) *T { return &v }

func TestNewListPager_FiresGet(t *testing.T) {
	ft := &fakeTransport{body: `{"value":[{"type":"ModelDeployment","name":"d1","modelName":"gpt-4o"}]}`}
	c := newTestClient(t, ft)

	page, err := c.NewListPager(nil).NextPage(context.Background())
	if err != nil {
		t.Fatalf("NextPage: %v", err)
	}
	if got := ft.gotReq.URL.Path; got != "/deployments" {
		t.Errorf("path = %s, want /deployments", got)
	}
	if got := ft.gotReq.URL.Query().Get("api-version"); got != "v1" {
		t.Errorf("api-version = %s, want v1", got)
	}
	if len(page.Value) != 1 || page.Value[0].ModelName != "gpt-4o" {
		t.Errorf("page = %+v, want one ModelDeployment with gpt-4o", page)
	}
}

func TestNewListPager_AppliesModelPublisherFilter(t *testing.T) {
	ft := &fakeTransport{body: `{"value":[]}`}
	c := newTestClient(t, ft)

	if _, err := c.NewListPager(&ListOptions{ModelPublisher: ptr("openai")}).NextPage(context.Background()); err != nil {
		t.Fatalf("NextPage: %v", err)
	}
	if got := ft.gotReq.URL.Query().Get("modelPublisher"); got != "openai" {
		t.Errorf("modelPublisher = %s, want openai", got)
	}
}

func TestGet_FiresGetByName(t *testing.T) {
	ft := &fakeTransport{body: `{"type":"ModelDeployment","name":"my-gpt4o","modelName":"gpt-4o"}`}
	c := newTestClient(t, ft)

	resp, err := c.Get(context.Background(), "my-gpt4o", nil)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got := ft.gotReq.URL.Path; got != "/deployments/my-gpt4o" {
		t.Errorf("path = %s, want /deployments/my-gpt4o", got)
	}
	if resp.Name != "my-gpt4o" {
		t.Errorf("Name = %q, want my-gpt4o", resp.Name)
	}
}

func TestGet_RejectsEmptyName(t *testing.T) {
	c := newTestClient(t, &fakeTransport{})
	if _, err := c.Get(context.Background(), "", nil); err == nil {
		t.Fatal("Get with empty name: want error")
	}
}
