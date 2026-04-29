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
	if c.Agents() == nil || c.Agents().Endpoint() != "https://example.test" {
		t.Errorf("Agents() endpoint = %q", c.Agents().Endpoint())
	}
	if c.EvaluationRules() == nil || c.EvaluationRules().Endpoint() != "https://example.test" {
		t.Errorf("EvaluationRules() endpoint = %q", c.EvaluationRules().Endpoint())
	}

	b := c.Beta()
	if b == nil || b.Endpoint() != "https://example.test" {
		t.Fatalf("Beta() endpoint = %q", b.Endpoint())
	}
	if b.Skills() == nil || b.Skills().Endpoint() != "https://example.test" {
		t.Errorf("Beta().Skills() endpoint = %q", b.Skills().Endpoint())
	}
	if b.Toolboxes() == nil || b.Toolboxes().Endpoint() != "https://example.test" {
		t.Errorf("Beta().Toolboxes() endpoint = %q", b.Toolboxes().Endpoint())
	}
	if b.Schedules() == nil || b.Schedules().Endpoint() != "https://example.test" {
		t.Errorf("Beta().Schedules() endpoint = %q", b.Schedules().Endpoint())
	}
	if b.RedTeams() == nil || b.RedTeams().Endpoint() != "https://example.test" {
		t.Errorf("Beta().RedTeams() endpoint = %q", b.RedTeams().Endpoint())
	}
	if b.MemoryStores() == nil || b.MemoryStores().Endpoint() != "https://example.test" {
		t.Errorf("Beta().MemoryStores() endpoint = %q", b.MemoryStores().Endpoint())
	}
	if b.Insights() == nil || b.Insights().Endpoint() != "https://example.test" {
		t.Errorf("Beta().Insights() endpoint = %q", b.Insights().Endpoint())
	}
	if b.Evaluators() == nil || b.Evaluators().Endpoint() != "https://example.test" {
		t.Errorf("Beta().Evaluators() endpoint = %q", b.Evaluators().Endpoint())
	}
	if b.EvaluationTaxonomies() == nil || b.EvaluationTaxonomies().Endpoint() != "https://example.test" {
		t.Errorf("Beta().EvaluationTaxonomies() endpoint = %q", b.EvaluationTaxonomies().Endpoint())
	}
	if b.Agents() == nil || b.Agents().Endpoint() != "https://example.test" {
		t.Errorf("Beta().Agents() endpoint = %q", b.Agents().Endpoint())
	}
}
