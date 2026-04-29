package connections

import (
	"context"
	"net/http"
	"testing"
)

func TestGet_FiresGetByName(t *testing.T) {
	ft := &fakeTransport{
		body: `{"name":"c1","type":"AzureOpenAI"}`,
	}
	c := newTestClient(t, ft)

	resp, err := c.Get(context.Background(), "c1", nil)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	if got, want := ft.gotReq.Method, http.MethodGet; got != want {
		t.Errorf("method = %s, want %s", got, want)
	}
	if got, want := ft.gotReq.URL.Path, "/connections/c1"; got != want {
		t.Errorf("path = %s, want %s", got, want)
	}
	if got, want := ft.gotReq.URL.Query().Get("api-version"), "v1"; got != want {
		t.Errorf("api-version = %s, want %s", got, want)
	}
	if resp.Name != "c1" || resp.Type != ConnectionTypeAzureOpenAI {
		t.Errorf("resp = %+v, want name=c1 type=AzureOpenAI", resp.Connection)
	}
}

func TestGet_RejectsEmptyName(t *testing.T) {
	c := newTestClient(t, &fakeTransport{})
	if _, err := c.Get(context.Background(), "", nil); err == nil {
		t.Fatal("Get with empty name: want error, got nil")
	}
}