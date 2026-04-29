package agents

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
)

// CreateOptions is the optional parameter set for Create.
type CreateOptions struct {
	FoundryFeatures    *string
	Metadata           map[string]string
	Description        *string
	BlueprintReference *AgentBlueprintReference
	AgentEndpoint      *AgentEndpoint
	AgentCard          *AgentCard
}

// createBody is the on-the-wire payload for POST /agents.
type createBody struct {
	Name               string                   `json:"name"`
	Metadata           map[string]string        `json:"metadata,omitempty"`
	Description        *string                  `json:"description,omitempty"`
	Definition         AgentDefinitionUnion     `json:"definition"`
	BlueprintReference *AgentBlueprintReference `json:"blueprint_reference,omitempty"`
	AgentEndpoint      *AgentEndpoint           `json:"agent_endpoint,omitempty"`
	AgentCard          *AgentCard               `json:"agent_card,omitempty"`
}

// Create creates a new agent.
func (c *Client) Create(ctx context.Context, name string, def AgentDefinitionUnion, opts *CreateOptions) (Agent, error) {
	if name == "" {
		return Agent{}, errors.New("agents.Create: name is required")
	}
	if def == nil {
		return Agent{}, errors.New("agents.Create: definition is required")
	}
	body := createBody{Name: name, Definition: def}
	var foundry *string
	if opts != nil {
		body.Metadata = opts.Metadata
		body.Description = opts.Description
		body.BlueprintReference = opts.BlueprintReference
		body.AgentEndpoint = opts.AgentEndpoint
		body.AgentCard = opts.AgentCard
		foundry = opts.FoundryFeatures
	}
	return doPostJSON[Agent](ctx, c, "/agents", body, foundry)
}

// UpdateOptions is the optional parameter set for Update.
type UpdateOptions struct {
	FoundryFeatures    *string
	Metadata           map[string]string
	Description        *string
	BlueprintReference *AgentBlueprintReference
}

// updateBody is the on-the-wire payload for POST /agents/{name}.
type updateBody struct {
	Metadata           map[string]string        `json:"metadata,omitempty"`
	Description        *string                  `json:"description,omitempty"`
	Definition         AgentDefinitionUnion     `json:"definition"`
	BlueprintReference *AgentBlueprintReference `json:"blueprint_reference,omitempty"`
}

// Update creates a new version of an existing agent. If the definition is
// unchanged the service returns the existing agent unchanged.
func (c *Client) Update(ctx context.Context, agentName string, def AgentDefinitionUnion, opts *UpdateOptions) (Agent, error) {
	if agentName == "" {
		return Agent{}, errors.New("agents.Update: agentName is required")
	}
	if def == nil {
		return Agent{}, errors.New("agents.Update: definition is required")
	}
	body := updateBody{Definition: def}
	var foundry *string
	if opts != nil {
		body.Metadata = opts.Metadata
		body.Description = opts.Description
		body.BlueprintReference = opts.BlueprintReference
		foundry = opts.FoundryFeatures
	}
	return doPostJSON[Agent](ctx, c, fmt.Sprintf("/agents/%s", agentName), body, foundry)
}

// CreateVersionOptions is the optional parameter set for CreateVersion.
type CreateVersionOptions struct {
	FoundryFeatures    *string
	Metadata           map[string]string
	Description        *string
	BlueprintReference *AgentBlueprintReference
}

// createVersionBody is the on-the-wire payload for POST /agents/{name}/versions.
type createVersionBody struct {
	Metadata           map[string]string        `json:"metadata,omitempty"`
	Description        *string                  `json:"description,omitempty"`
	Definition         AgentDefinitionUnion     `json:"definition"`
	BlueprintReference *AgentBlueprintReference `json:"blueprint_reference,omitempty"`
}

// CreateVersion creates a new version of an existing agent.
func (c *Client) CreateVersion(ctx context.Context, agentName string, def AgentDefinitionUnion, opts *CreateVersionOptions) (AgentVersion, error) {
	if agentName == "" {
		return AgentVersion{}, errors.New("agents.CreateVersion: agentName is required")
	}
	if def == nil {
		return AgentVersion{}, errors.New("agents.CreateVersion: definition is required")
	}
	body := createVersionBody{Definition: def}
	var foundry *string
	if opts != nil {
		body.Metadata = opts.Metadata
		body.Description = opts.Description
		body.BlueprintReference = opts.BlueprintReference
		foundry = opts.FoundryFeatures
	}
	return doPostJSON[AgentVersion](ctx, c, fmt.Sprintf("/agents/%s/versions", agentName), body, foundry)
}

// DeleteOptions is the optional parameter set for Delete.
type DeleteOptions struct{}

// Delete removes an agent (and all its versions).
func (c *Client) Delete(ctx context.Context, agentName string, _ *DeleteOptions) (DeleteAgentResponse, error) {
	if agentName == "" {
		return DeleteAgentResponse{}, errors.New("agents.Delete: agentName is required")
	}
	req, err := runtime.NewRequest(ctx, http.MethodDelete, c.endpoint+"/agents/"+agentName)
	if err != nil {
		return DeleteAgentResponse{}, err
	}
	c.setAPIVersion(req)
	return doJSON[DeleteAgentResponse](c, req)
}

// DeleteVersionOptions is the optional parameter set for DeleteVersion.
type DeleteVersionOptions struct{}

// DeleteVersion removes a specific version of an agent.
func (c *Client) DeleteVersion(ctx context.Context, agentName, agentVersion string, _ *DeleteVersionOptions) (DeleteAgentVersionResponse, error) {
	if agentName == "" || agentVersion == "" {
		return DeleteAgentVersionResponse{}, errors.New("agents.DeleteVersion: agentName and agentVersion are required")
	}
	req, err := runtime.NewRequest(ctx, http.MethodDelete,
		fmt.Sprintf("%s/agents/%s/versions/%s", c.endpoint, agentName, agentVersion))
	if err != nil {
		return DeleteAgentVersionResponse{}, err
	}
	c.setAPIVersion(req)
	return doJSON[DeleteAgentVersionResponse](c, req)
}

// --- shared write helper ---

// doPostJSON marshals body, POSTs to path, and decodes the JSON response into T.
// If foundryFeatures is non-nil, sets the foundry-features header to
// "<value>,AgentEndpoints=V1Preview" per the TS reference.
func doPostJSON[T any](ctx context.Context, c *Client, path string, body any, foundryFeatures *string) (T, error) {
	var zero T
	payload, err := json.Marshal(body)
	if err != nil {
		return zero, fmt.Errorf("agents: marshal body: %w", err)
	}
	req, err := runtime.NewRequest(ctx, http.MethodPost, c.endpoint+path)
	if err != nil {
		return zero, err
	}
	c.setAPIVersion(req)
	if foundryFeatures != nil {
		req.Raw().Header.Set("foundry-features", *foundryFeatures+",AgentEndpoints=V1Preview")
	}
	if err := req.SetBody(byteSeeker{bytes.NewReader(payload)}, "application/json"); err != nil {
		return zero, err
	}
	req.Raw().Header.Set("Accept", "application/json")
	return doJSON[T](c, req)
}

type byteSeeker struct{ *bytes.Reader }

func (byteSeeker) Close() error { return nil }
