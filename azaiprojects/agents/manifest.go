package agents

import (
	"context"
	"errors"
	"fmt"
)

// CreateFromManifestOptions is the optional parameter set for CreateFromManifest.
type CreateFromManifestOptions struct {
	Metadata    map[string]string
	Description *string
}

// CreateFromManifest creates an agent from a published manifest.
// POST /agents:import.
func (c *Client) CreateFromManifest(ctx context.Context, name, manifestID string, parameterValues map[string]any, opts *CreateFromManifestOptions) (Agent, error) {
	if name == "" {
		return Agent{}, errors.New("agents.CreateFromManifest: name is required")
	}
	if manifestID == "" {
		return Agent{}, errors.New("agents.CreateFromManifest: manifestID is required")
	}
	if parameterValues == nil {
		parameterValues = map[string]any{}
	}
	body := agentManifestBody{
		Name:            name,
		ManifestID:      manifestID,
		ParameterValues: parameterValues,
	}
	if opts != nil {
		body.Metadata = opts.Metadata
		if opts.Description != nil {
			body.Description = *opts.Description
		}
	}
	return doPostJSON[Agent](ctx, c, "/agents:import", body, nil)
}

// UpdateFromManifestOptions is the optional parameter set for UpdateFromManifest.
type UpdateFromManifestOptions struct {
	Metadata    map[string]string
	Description *string
}

// UpdateFromManifest updates an agent from a published manifest by adding a
// new version when the definition changes. POST /agents/{name}/import.
func (c *Client) UpdateFromManifest(ctx context.Context, agentName, manifestID string, parameterValues map[string]any, opts *UpdateFromManifestOptions) (Agent, error) {
	if agentName == "" {
		return Agent{}, errors.New("agents.UpdateFromManifest: agentName is required")
	}
	if manifestID == "" {
		return Agent{}, errors.New("agents.UpdateFromManifest: manifestID is required")
	}
	if parameterValues == nil {
		parameterValues = map[string]any{}
	}
	body := agentVersionManifestBody{
		ManifestID:      manifestID,
		ParameterValues: parameterValues,
	}
	if opts != nil {
		body.Metadata = opts.Metadata
		if opts.Description != nil {
			body.Description = *opts.Description
		}
	}
	return doPostJSON[Agent](ctx, c, fmt.Sprintf("/agents/%s/import", agentName), body, nil)
}

// CreateVersionFromManifestOptions is the optional parameter set for CreateVersionFromManifest.
type CreateVersionFromManifestOptions struct {
	Metadata    map[string]string
	Description *string
}

// CreateVersionFromManifest creates a new version of an existing agent from a
// published manifest. POST /agents/{name}/versions:import.
func (c *Client) CreateVersionFromManifest(ctx context.Context, agentName, manifestID string, parameterValues map[string]any, opts *CreateVersionFromManifestOptions) (AgentVersion, error) {
	if agentName == "" {
		return AgentVersion{}, errors.New("agents.CreateVersionFromManifest: agentName is required")
	}
	if manifestID == "" {
		return AgentVersion{}, errors.New("agents.CreateVersionFromManifest: manifestID is required")
	}
	if parameterValues == nil {
		parameterValues = map[string]any{}
	}
	body := agentVersionManifestBody{
		ManifestID:      manifestID,
		ParameterValues: parameterValues,
	}
	if opts != nil {
		body.Metadata = opts.Metadata
		if opts.Description != nil {
			body.Description = *opts.Description
		}
	}
	return doPostJSON[AgentVersion](ctx, c, fmt.Sprintf("/agents/%s/versions:import", agentName), body, nil)
}
