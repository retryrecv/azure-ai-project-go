package connections

import (
	"context"
	"net/http"
	"testing"
)

func TestGetWithCredentials_FiresGetByName(t *testing.T) {
	ft := &fakeTransport{
		body: `{"name":"c1","type":"AzureOpenAI","credentials":{"type":"ApiKey","key":"k"}}`,
	}
	c := newTestClient(t, ft)

	resp, err := c.GetWithCredentials(context.Background(), "c1", nil)
	if err != nil {
		t.Fatalf("GetWithCredentials: %v", err)
	}

	if got, want := ft.gotReq.Method, http.MethodGet; got != want {
		t.Errorf("method = %s, want %s", got, want)
	}
	if got, want := ft.gotReq.URL.Path, "/connections/c1/getConnectionWithCredentials"; got != want {
		t.Errorf("path = %s, want %s", got, want)
	}
	if resp.Credentials.Type != CredentialTypeAPIKey {
		t.Errorf("Credentials.Type = %q, want %q", resp.Credentials.Type, CredentialTypeAPIKey)
	}
	if resp.Credentials.APIKey != "k" {
		t.Errorf("Credentials.APIKey = %q, want k", resp.Credentials.APIKey)
	}
}

func TestGetWithCredentials_RejectsEmptyName(t *testing.T) {
	c := newTestClient(t, &fakeTransport{})
	if _, err := c.GetWithCredentials(context.Background(), "", nil); err == nil {
		t.Fatal("GetWithCredentials with empty name: want error, got nil")
	}
}