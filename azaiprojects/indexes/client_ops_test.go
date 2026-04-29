package indexes

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
)

func TestNewListVersionsPager_FiresGet(t *testing.T) {
	ft := &fakeTransport{body: `{"value":[{"name":"my-index","version":"1.0","type":"AzureSearch"}]}`}
	c := newTestClient(t, ft)

	if _, err := c.NewListVersionsPager("my-index", nil).NextPage(context.Background()); err != nil {
		t.Fatalf("NextPage: %v", err)
	}
	if got := ft.gotReq.URL.Path; got != "/indexes/my-index/versions" {
		t.Errorf("path = %s, want /indexes/my-index/versions", got)
	}
	if got := ft.gotReq.URL.Query().Get("api-version"); got != "v1" {
		t.Errorf("api-version = %s, want v1", got)
	}
}

func TestGet_FiresGet(t *testing.T) {
	ft := &fakeTransport{body: `{"name":"my-index","version":"1.0","type":"AzureSearch","indexName":"docs","connectionName":"sc"}`}
	c := newTestClient(t, ft)

	resp, err := c.Get(context.Background(), "my-index", "1.0", nil)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got := ft.gotReq.URL.Path; got != "/indexes/my-index/versions/1.0" {
		t.Errorf("path = %s, want /indexes/my-index/versions/1.0", got)
	}
	if resp.Name != "my-index" || resp.Type != IndexTypeAzureSearch {
		t.Errorf("got = %+v", resp)
	}

	// Confirm Raw lets the caller decode into the concrete type.
	var concrete AzureAISearchIndex
	if err := json.Unmarshal(resp.Raw, &concrete); err != nil {
		t.Fatalf("Raw decode: %v", err)
	}
	if concrete.IndexName != "docs" {
		t.Errorf("concrete.IndexName = %q, want docs", concrete.IndexName)
	}
}

func TestCreateOrUpdate_FiresPatchWithMergeJSON(t *testing.T) {
	ft := &fakeTransport{
		body:   `{"name":"x","version":"1.0","type":"AzureSearch","indexName":"i","connectionName":"c"}`,
		status: http.StatusCreated,
	}
	c := newTestClient(t, ft)

	body := AzureAISearchIndex{
		Index:          Index{Name: "x", Version: "1.0", Type: IndexTypeAzureSearch},
		IndexName:      "i",
		ConnectionName: "c",
	}
	resp, err := c.CreateOrUpdate(context.Background(), "x", "1.0", body, nil)
	if err != nil {
		t.Fatalf("CreateOrUpdate: %v", err)
	}
	if got := ft.gotReq.Method; got != http.MethodPatch {
		t.Errorf("method = %s, want PATCH", got)
	}
	if got := ft.gotReq.URL.Path; got != "/indexes/x/versions/1.0" {
		t.Errorf("path = %s, want /indexes/x/versions/1.0", got)
	}
	if got, want := ft.gotReq.Header.Get("Content-Type"), "application/merge-patch+json"; got != want {
		t.Errorf("Content-Type = %q, want %q", got, want)
	}

	var sent map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(ft.gotBody), &sent); err != nil {
		t.Fatalf("request body unmarshal: %v (body=%q)", err, ft.gotBody)
	}
	if sent["type"] != "AzureSearch" {
		t.Errorf("body.type = %v, want AzureSearch", sent["type"])
	}
	if sent["indexName"] != "i" {
		t.Errorf("body.indexName = %v, want i", sent["indexName"])
	}
	if resp.Type != IndexTypeAzureSearch {
		t.Errorf("response.Type = %q, want AzureSearch", resp.Type)
	}
}

func TestDelete_204IsSuccess(t *testing.T) {
	ft := &fakeTransport{status: http.StatusNoContent, body: ""}
	c := newTestClient(t, ft)

	if _, err := c.Delete(context.Background(), "x", "1.0", nil); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if got := ft.gotReq.Method; got != http.MethodDelete {
		t.Errorf("method = %s, want DELETE", got)
	}
	if got := ft.gotReq.URL.Path; got != "/indexes/x/versions/1.0" {
		t.Errorf("path = %s, want /indexes/x/versions/1.0", got)
	}
}

func TestDelete_404IsResponseError(t *testing.T) {
	ft := &fakeTransport{status: http.StatusNotFound, body: `{"error":{"code":"NotFound","message":"missing"}}`}
	c := newTestClient(t, ft)

	_, err := c.Delete(context.Background(), "x", "1.0", nil)
	if err == nil {
		t.Fatal("Delete with 404: want error, got nil")
	}
	var rerr *azcore.ResponseError
	if !errors.As(err, &rerr) {
		t.Fatalf("Delete error = %T %v, want *azcore.ResponseError", err, err)
	}
	if rerr.StatusCode != http.StatusNotFound {
		t.Errorf("StatusCode = %d, want 404", rerr.StatusCode)
	}
}
