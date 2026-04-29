package datasets

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

type fakeCred struct{}

func (fakeCred) GetToken(_ context.Context, _ policy.TokenRequestOptions) (azcore.AccessToken, error) {
	return azcore.AccessToken{Token: "fake", ExpiresOn: time.Now().Add(time.Hour)}, nil
}

type fakeTransport struct {
	gotReq  *http.Request
	gotBody []byte
	body    string
	status  int
}

func (f *fakeTransport) Do(req *http.Request) (*http.Response, error) {
	f.gotReq = req
	if req.Body != nil {
		f.gotBody, _ = io.ReadAll(req.Body)
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
	ft := &fakeTransport{body: `{"value":[{"name":"d1","version":"1.0","type":"uri_file","dataUri":"u"}]}`}
	c := newTestClient(t, ft)

	if _, err := c.NewListPager(nil).NextPage(context.Background()); err != nil {
		t.Fatalf("NextPage: %v", err)
	}
	if ft.gotReq.URL.Path != "/datasets" {
		t.Errorf("path = %s, want /datasets", ft.gotReq.URL.Path)
	}
	if ft.gotReq.URL.Query().Get("api-version") != "v1" {
		t.Errorf("api-version = %s, want v1", ft.gotReq.URL.Query().Get("api-version"))
	}
}

func TestNewListVersionsPager_FiresGet(t *testing.T) {
	ft := &fakeTransport{body: `{"value":[]}`}
	c := newTestClient(t, ft)
	if _, err := c.NewListVersionsPager("d1", nil).NextPage(context.Background()); err != nil {
		t.Fatalf("NextPage: %v", err)
	}
	if ft.gotReq.URL.Path != "/datasets/d1/versions" {
		t.Errorf("path = %s, want /datasets/d1/versions", ft.gotReq.URL.Path)
	}
}

func TestGet_FiresGet(t *testing.T) {
	ft := &fakeTransport{body: `{"name":"d1","version":"1.0","type":"uri_file","dataUri":"u"}`}
	c := newTestClient(t, ft)
	resp, err := c.Get(context.Background(), "d1", "1.0", nil)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if ft.gotReq.URL.Path != "/datasets/d1/versions/1.0" {
		t.Errorf("path = %s", ft.gotReq.URL.Path)
	}
	if resp.Name != "d1" || resp.Type != DatasetTypeURIFile {
		t.Errorf("got = %+v", resp)
	}
}

func TestCreateOrUpdate_FiresPatchWithMergeJSON(t *testing.T) {
	ft := &fakeTransport{
		body:   `{"name":"d1","version":"1.0","type":"uri_file","dataUri":"https://x"}`,
		status: http.StatusOK,
	}
	c := newTestClient(t, ft)

	body := FileDatasetVersion{
		DatasetVersion: DatasetVersion{
			Name: "d1", Version: "1.0", Type: DatasetTypeURIFile, DataURI: "https://x",
		},
	}
	if _, err := c.CreateOrUpdate(context.Background(), "d1", "1.0", body, nil); err != nil {
		t.Fatalf("CreateOrUpdate: %v", err)
	}
	if ft.gotReq.Method != http.MethodPatch {
		t.Errorf("method = %s, want PATCH", ft.gotReq.Method)
	}
	if got := ft.gotReq.Header.Get("Content-Type"); got != "application/merge-patch+json" {
		t.Errorf("Content-Type = %q", got)
	}
	var sent map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(ft.gotBody), &sent); err != nil {
		t.Fatalf("body unmarshal: %v", err)
	}
	if sent["type"] != "uri_file" {
		t.Errorf("body.type = %v, want uri_file", sent["type"])
	}
}

func TestDelete_204IsSuccess(t *testing.T) {
	ft := &fakeTransport{status: http.StatusNoContent}
	c := newTestClient(t, ft)
	if _, err := c.Delete(context.Background(), "d1", "1.0", nil); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if ft.gotReq.Method != http.MethodDelete {
		t.Errorf("method = %s, want DELETE", ft.gotReq.Method)
	}
}

func TestPendingUpload_FiresPostWithBody(t *testing.T) {
	ft := &fakeTransport{
		body: `{
			"pendingUploadId":"u1",
			"pendingUploadType":"BlobReference",
			"blobReference":{"blobUri":"https://blob/x","storageAccountArmId":"sa","credential":{"sasUri":"https://blob/x?sv","type":"SAS"}}
		}`,
	}
	c := newTestClient(t, ft)
	resp, err := c.PendingUpload(context.Background(), "d1", "1.0",
		PendingUploadRequest{ConnectionName: "sc", PendingUploadType: PendingUploadTypeBlobReference}, nil)
	if err != nil {
		t.Fatalf("PendingUpload: %v", err)
	}
	if ft.gotReq.Method != http.MethodPost {
		t.Errorf("method = %s, want POST", ft.gotReq.Method)
	}
	if ft.gotReq.URL.Path != "/datasets/d1/versions/1.0/startPendingUpload" {
		t.Errorf("path = %s", ft.gotReq.URL.Path)
	}
	var sent map[string]any
	if err := json.Unmarshal(ft.gotBody, &sent); err != nil {
		t.Fatalf("body: %v", err)
	}
	if sent["connectionName"] != "sc" || sent["pendingUploadType"] != "BlobReference" {
		t.Errorf("body = %+v", sent)
	}
	if resp.BlobReference.BlobURI != "https://blob/x" {
		t.Errorf("BlobURI = %q", resp.BlobReference.BlobURI)
	}
}

func TestGetCredentials_FiresPost(t *testing.T) {
	ft := &fakeTransport{
		body: `{"blobReference":{"blobUri":"https://blob/x","storageAccountArmId":"sa","credential":{"sasUri":"https://blob/x?sv","type":"SAS"}}}`,
	}
	c := newTestClient(t, ft)
	cred, err := c.GetCredentials(context.Background(), "d1", "1.0", nil)
	if err != nil {
		t.Fatalf("GetCredentials: %v", err)
	}
	if ft.gotReq.Method != http.MethodPost {
		t.Errorf("method = %s, want POST", ft.gotReq.Method)
	}
	if ft.gotReq.URL.Path != "/datasets/d1/versions/1.0/credentials" {
		t.Errorf("path = %s", ft.gotReq.URL.Path)
	}
	if cred.BlobReference.Credential.SASURI != "https://blob/x?sv" {
		t.Errorf("SASURI = %q", cred.BlobReference.Credential.SASURI)
	}
}
