package azaiprojects

import (
	"context"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
)

type fakeCred struct{}

func (fakeCred) GetToken(_ context.Context, _ policy.TokenRequestOptions) (azcore.AccessToken, error) {
	return azcore.AccessToken{Token: "fake", ExpiresOn: time.Now().Add(time.Hour)}, nil
}

func TestNewClient_Defaults(t *testing.T) {
	c, err := NewClient("https://example.test", fakeCred{}, nil)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if got, want := c.Endpoint(), "https://example.test"; got != want {
		t.Errorf("Endpoint = %q, want %q", got, want)
	}
	if got, want := c.APIVersion(), "v1"; got != want {
		t.Errorf("APIVersion = %q, want %q", got, want)
	}
}

func TestNewClient_RespectsAPIVersionOverride(t *testing.T) {
	c, err := NewClient("https://example.test", fakeCred{}, &ClientOptions{APIVersion: "2025-01-01"})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if got, want := c.APIVersion(), "2025-01-01"; got != want {
		t.Errorf("APIVersion = %q, want %q", got, want)
	}
}

func TestNewClient_RejectsEmptyEndpoint(t *testing.T) {
	if _, err := NewClient("", fakeCred{}, nil); err == nil {
		t.Fatal("NewClient with empty endpoint: want error, got nil")
	}
}

func TestNewClient_RejectsNilCredential(t *testing.T) {
	if _, err := NewClient("https://example.test", nil, nil); err == nil {
		t.Fatal("NewClient with nil credential: want error, got nil")
	}
}
