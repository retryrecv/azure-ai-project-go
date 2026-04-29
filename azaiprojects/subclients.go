package azaiprojects

import (
	"github.com/sambo/ai-projects-go/azaiprojects/agents"
	"github.com/sambo/ai-projects-go/azaiprojects/connections"
	"github.com/sambo/ai-projects-go/azaiprojects/datasets"
	"github.com/sambo/ai-projects-go/azaiprojects/deployments"
	"github.com/sambo/ai-projects-go/azaiprojects/evaluationrules"
	"github.com/sambo/ai-projects-go/azaiprojects/indexes"
)

// Connections returns the Connections operation group.
//
// The returned client shares the parent's pipeline, endpoint, and api-version.
// Calling this multiple times returns a fresh sub-client each time; cache it
// at the call site if you want to reuse it.
func (c *Client) Connections() *connections.Client {
	return connections.NewClientFromPipeline(c.endpoint, c.apiVersion, c.pl)
}

// Deployments returns the Deployments operation group.
func (c *Client) Deployments() *deployments.Client {
	return deployments.NewClientFromPipeline(c.endpoint, c.apiVersion, c.pl)
}

// Indexes returns the Indexes operation group.
func (c *Client) Indexes() *indexes.Client {
	return indexes.NewClientFromPipeline(c.endpoint, c.apiVersion, c.pl)
}

// Datasets returns the Datasets operation group.
func (c *Client) Datasets() *datasets.Client {
	return datasets.NewClientFromPipeline(c.endpoint, c.apiVersion, c.pl)
}

// Agents returns the Agents operation group.
func (c *Client) Agents() *agents.Client {
	return agents.NewClientFromPipeline(c.endpoint, c.apiVersion, c.pl)
}

// EvaluationRules returns the EvaluationRules operation group.
func (c *Client) EvaluationRules() *evaluationrules.Client {
	return evaluationrules.NewClientFromPipeline(c.endpoint, c.apiVersion, c.pl)
}
