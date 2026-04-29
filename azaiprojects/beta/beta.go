// Package beta exposes the project.beta operation groups.
//
// Use azaiprojects.Client.Beta() to obtain a BetaOperations container that
// provides accessors for each beta sub-client (skills, toolboxes, schedules,
// redteams, memorystores, insights, evaluators, evaluationtaxonomies, agents).
//
// Each accessor returns a fresh sub-client that shares the parent's pipeline,
// endpoint, and api-version. Cache at the call site for repeated use.
package beta

import (
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"

	"github.com/sambo/ai-projects-go/azaiprojects/beta/agents"
	"github.com/sambo/ai-projects-go/azaiprojects/beta/evaluationtaxonomies"
	"github.com/sambo/ai-projects-go/azaiprojects/beta/evaluators"
	"github.com/sambo/ai-projects-go/azaiprojects/beta/insights"
	"github.com/sambo/ai-projects-go/azaiprojects/beta/memorystores"
	"github.com/sambo/ai-projects-go/azaiprojects/beta/redteams"
	"github.com/sambo/ai-projects-go/azaiprojects/beta/schedules"
	"github.com/sambo/ai-projects-go/azaiprojects/beta/skills"
	"github.com/sambo/ai-projects-go/azaiprojects/beta/toolboxes"
)

// Operations is the beta operation group container.
type Operations struct {
	endpoint   string
	apiVersion string
	pl         runtime.Pipeline
}

// New constructs a beta Operations container that shares pipeline/endpoint/apiVersion.
func New(endpoint, apiVersion string, pl runtime.Pipeline) *Operations {
	return &Operations{endpoint: endpoint, apiVersion: apiVersion, pl: pl}
}

// Endpoint returns the configured service endpoint.
func (o *Operations) Endpoint() string { return o.endpoint }

// Skills returns the beta.skills sub-client.
func (o *Operations) Skills() *skills.Client {
	return skills.NewClientFromPipeline(o.endpoint, o.apiVersion, o.pl)
}

// Toolboxes returns the beta.toolboxes sub-client.
func (o *Operations) Toolboxes() *toolboxes.Client {
	return toolboxes.NewClientFromPipeline(o.endpoint, o.apiVersion, o.pl)
}

// Schedules returns the beta.schedules sub-client.
func (o *Operations) Schedules() *schedules.Client {
	return schedules.NewClientFromPipeline(o.endpoint, o.apiVersion, o.pl)
}

// RedTeams returns the beta.redTeams sub-client.
func (o *Operations) RedTeams() *redteams.Client {
	return redteams.NewClientFromPipeline(o.endpoint, o.apiVersion, o.pl)
}

// MemoryStores returns the beta.memoryStores sub-client.
func (o *Operations) MemoryStores() *memorystores.Client {
	return memorystores.NewClientFromPipeline(o.endpoint, o.apiVersion, o.pl)
}

// Insights returns the beta.insights sub-client.
func (o *Operations) Insights() *insights.Client {
	return insights.NewClientFromPipeline(o.endpoint, o.apiVersion, o.pl)
}

// Evaluators returns the beta.evaluators sub-client.
func (o *Operations) Evaluators() *evaluators.Client {
	return evaluators.NewClientFromPipeline(o.endpoint, o.apiVersion, o.pl)
}

// EvaluationTaxonomies returns the beta.evaluationTaxonomies sub-client.
func (o *Operations) EvaluationTaxonomies() *evaluationtaxonomies.Client {
	return evaluationtaxonomies.NewClientFromPipeline(o.endpoint, o.apiVersion, o.pl)
}

// Agents returns the beta.agents sub-client.
func (o *Operations) Agents() *agents.Client {
	return agents.NewClientFromPipeline(o.endpoint, o.apiVersion, o.pl)
}
