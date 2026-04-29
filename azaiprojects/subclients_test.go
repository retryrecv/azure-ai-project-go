package azaiprojects

import "testing"

func TestSubClientAccessors(t *testing.T) {
	c, err := NewClient("https://example.test", fakeCred{}, nil)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	if c.Connections() == nil || c.Connections().Endpoint() != "https://example.test" {
		t.Errorf("Connections() endpoint = %q, want https://example.test", c.Connections().Endpoint())
	}
	if c.Deployments() == nil || c.Deployments().Endpoint() != "https://example.test" {
		t.Errorf("Deployments() endpoint = %q", c.Deployments().Endpoint())
	}
	if c.Indexes() == nil || c.Indexes().Endpoint() != "https://example.test" {
		t.Errorf("Indexes() endpoint = %q", c.Indexes().Endpoint())
	}
	if c.Datasets() == nil || c.Datasets().Endpoint() != "https://example.test" {
		t.Errorf("Datasets() endpoint = %q", c.Datasets().Endpoint())
	}
}
