package connections

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
)

// scriptedTransport returns a sequence of canned responses, one per request.
type scriptedTransport struct {
	calls []scriptedCall
	idx   int
}

type scriptedCall struct {
	wantPath string
	body     string
	status   int
}

func (s *scriptedTransport) Do(req *http.Request) (*http.Response, error) {
	if s.idx >= len(s.calls) {
		return nil, &unexpectedCallError{path: req.URL.Path}
	}
	call := s.calls[s.idx]
	s.idx++
	status := call.status
	if status == 0 {
		status = http.StatusOK
	}
	return &http.Response{
		Status:     http.StatusText(status),
		StatusCode: status,
		Body:       io.NopCloser(bytes.NewBufferString(call.body)),
		Header:     make(http.Header),
		Request:    req,
	}, nil
}

type unexpectedCallError struct{ path string }

func (e *unexpectedCallError) Error() string { return "scriptedTransport: unexpected call to " + e.path }

func newScriptedClient(t *testing.T, calls []scriptedCall) (*Client, *scriptedTransport) {
	t.Helper()
	st := &scriptedTransport{calls: calls}
	c, err := NewClient("https://example.test", fakeCred{}, &ClientOptions{
		ClientOptions: azcore.ClientOptions{Transport: st},
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return c, st
}

func TestGetDefault_NoCredentialsFiresOnlyListWithFilters(t *testing.T) {
	c, st := newScriptedClient(t, []scriptedCall{
		{
			wantPath: "/connections",
			body:     `{"value":[{"name":"default-c","type":"AzureOpenAI","isDefault":true}]}`,
		},
	})

	resp, err := c.GetDefault(context.Background(), ConnectionTypeAzureOpenAI, nil)
	if err != nil {
		t.Fatalf("GetDefault: %v", err)
	}
	if resp.Name != "default-c" {
		t.Errorf("Name = %q, want default-c", resp.Name)
	}
	if st.idx != 1 {
		t.Errorf("requests sent = %d, want 1 (no follow-up)", st.idx)
	}
}

func TestGetDefault_WithCredentialsFiresFollowUp(t *testing.T) {
	c, st := newScriptedClient(t, []scriptedCall{
		{
			wantPath: "/connections",
			body:     `{"value":[{"name":"default-c","type":"AzureOpenAI","isDefault":true}]}`,
		},
		{
			wantPath: "/connections/default-c/getConnectionWithCredentials",
			body:     `{"name":"default-c","type":"AzureOpenAI","credentials":{"type":"ApiKey","key":"k"}}`,
		},
	})

	resp, err := c.GetDefault(context.Background(), ConnectionTypeAzureOpenAI, &GetDefaultOptions{IncludeCredentials: true})
	if err != nil {
		t.Fatalf("GetDefault: %v", err)
	}
	if st.idx != 2 {
		t.Fatalf("requests sent = %d, want 2", st.idx)
	}
	if resp.Credentials.Type != CredentialTypeAPIKey || resp.Credentials.APIKey != "k" {
		t.Errorf("Credentials = %+v, want ApiKey/k", resp.Credentials)
	}
}

func TestGetDefault_NoMatchReturnsError(t *testing.T) {
	c, _ := newScriptedClient(t, []scriptedCall{
		{wantPath: "/connections", body: `{"value":[]}`},
	})
	if _, err := c.GetDefault(context.Background(), ConnectionTypeAzureOpenAI, nil); err == nil {
		t.Fatal("GetDefault with empty list: want error, got nil")
	}
}

func TestGetDefault_RejectsEmptyType(t *testing.T) {
	c := newTestClient(t, &fakeTransport{})
	if _, err := c.GetDefault(context.Background(), "", nil); err == nil {
		t.Fatal("GetDefault with empty type: want error, got nil")
	}
}