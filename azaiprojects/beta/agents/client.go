package agents

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
)

const (
	moduleName        = "azaiprojects/beta/agents"
	moduleVersion     = "0.1.0"
	defaultScope      = "https://ai.azure.com/.default"
	defaultAPIVer     = "v1"
	hostedOnly        = "HostedAgents=V1Preview"
	hostedAndEndpoint = "HostedAgents=V1Preview,AgentEndpoints=V1Preview"
)

// Client provides the beta.agents operation group.
type Client struct {
	endpoint   string
	apiVersion string
	pl         runtime.Pipeline
}

// ClientOptions configures the agents client.
type ClientOptions struct {
	azcore.ClientOptions
	APIVersion string
}

// NewClient constructs an agents client targeting endpoint.
func NewClient(endpoint string, cred azcore.TokenCredential, opts *ClientOptions) (*Client, error) {
	if endpoint == "" {
		return nil, errors.New("agents: endpoint is required")
	}
	if cred == nil {
		return nil, errors.New("agents: cred is required")
	}
	if opts == nil {
		opts = &ClientOptions{}
	}
	apiVersion := opts.APIVersion
	if apiVersion == "" {
		apiVersion = defaultAPIVer
	}
	bearer := runtime.NewBearerTokenPolicy(cred, []string{defaultScope}, nil)
	pl := runtime.NewPipeline(moduleName, moduleVersion,
		runtime.PipelineOptions{PerRetry: []policy.Policy{bearer}},
		&opts.ClientOptions,
	)
	return &Client{endpoint: endpoint, apiVersion: apiVersion, pl: pl}, nil
}

// NewClientFromPipeline reuses an existing pipeline.
func NewClientFromPipeline(endpoint, apiVersion string, pl runtime.Pipeline) *Client {
	if apiVersion == "" {
		apiVersion = defaultAPIVer
	}
	return &Client{endpoint: endpoint, apiVersion: apiVersion, pl: pl}
}

// Endpoint returns the configured service endpoint.
func (c *Client) Endpoint() string { return c.endpoint }

// PatchAgentOptions is the optional parameter set for PatchAgent.
type PatchAgentOptions struct{}

// PatchAgent updates an agent endpoint.
// PATCH /agents/{agent_name} (merge-patch+json) returns 200.
func (c *Client) PatchAgent(ctx context.Context, agentName string, body PatchAgentBody, _ *PatchAgentOptions) (Agent, error) {
	if agentName == "" {
		return Agent{}, errors.New("agents.PatchAgent: agentName is required")
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return Agent{}, fmt.Errorf("agents.PatchAgent: marshal: %w", err)
	}
	respBody, err := c.jsonRequest(ctx, http.MethodPatch,
		fmt.Sprintf("%s/agents/%s", c.endpoint, agentName), payload,
		"application/merge-patch+json", hostedAndEndpoint, nil, http.StatusOK)
	if err != nil {
		return Agent{}, err
	}
	var out Agent
	if err := json.Unmarshal(respBody, &out); err != nil {
		return Agent{}, fmt.Errorf("agents.PatchAgent: decode: %w", err)
	}
	return out, nil
}

// CreateSessionOptions is the optional parameter set for CreateSession.
type CreateSessionOptions struct{}

// CreateSession creates a session for the given agent endpoint.
// POST /agents/{agent_name}/endpoint/sessions returns 201.
// Sets x-session-isolation-key from isolationKey.
func (c *Client) CreateSession(ctx context.Context, agentName, isolationKey string, body CreateSessionBody, _ *CreateSessionOptions) (AgentSessionResource, error) {
	if agentName == "" {
		return AgentSessionResource{}, errors.New("agents.CreateSession: agentName is required")
	}
	if isolationKey == "" {
		return AgentSessionResource{}, errors.New("agents.CreateSession: isolationKey is required")
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return AgentSessionResource{}, fmt.Errorf("agents.CreateSession: marshal: %w", err)
	}
	respBody, err := c.jsonRequest(ctx, http.MethodPost,
		fmt.Sprintf("%s/agents/%s/endpoint/sessions", c.endpoint, agentName), payload,
		"application/json", hostedAndEndpoint,
		map[string]string{"x-session-isolation-key": isolationKey}, http.StatusCreated)
	if err != nil {
		return AgentSessionResource{}, err
	}
	var out AgentSessionResource
	if err := json.Unmarshal(respBody, &out); err != nil {
		return AgentSessionResource{}, fmt.Errorf("agents.CreateSession: decode: %w", err)
	}
	return out, nil
}

// GetSessionOptions is the optional parameter set for GetSession.
type GetSessionOptions struct{}

// GetSession retrieves a session by ID.
func (c *Client) GetSession(ctx context.Context, agentName, sessionID string, _ *GetSessionOptions) (AgentSessionResource, error) {
	if agentName == "" {
		return AgentSessionResource{}, errors.New("agents.GetSession: agentName is required")
	}
	if sessionID == "" {
		return AgentSessionResource{}, errors.New("agents.GetSession: sessionID is required")
	}
	respBody, err := c.jsonRequest(ctx, http.MethodGet,
		fmt.Sprintf("%s/agents/%s/endpoint/sessions/%s", c.endpoint, agentName, sessionID),
		nil, "", hostedAndEndpoint, nil, http.StatusOK)
	if err != nil {
		return AgentSessionResource{}, err
	}
	var out AgentSessionResource
	if err := json.Unmarshal(respBody, &out); err != nil {
		return AgentSessionResource{}, fmt.Errorf("agents.GetSession: decode: %w", err)
	}
	return out, nil
}

// DeleteSessionOptions is the optional parameter set for DeleteSession.
type DeleteSessionOptions struct{}

// DeleteSessionResponse is the (empty) response of DeleteSession.
type DeleteSessionResponse struct{}

// DeleteSession deletes a session.
// DELETE returns 204. Sets x-session-isolation-key.
func (c *Client) DeleteSession(ctx context.Context, agentName, sessionID, isolationKey string, _ *DeleteSessionOptions) (DeleteSessionResponse, error) {
	if agentName == "" {
		return DeleteSessionResponse{}, errors.New("agents.DeleteSession: agentName is required")
	}
	if sessionID == "" {
		return DeleteSessionResponse{}, errors.New("agents.DeleteSession: sessionID is required")
	}
	if isolationKey == "" {
		return DeleteSessionResponse{}, errors.New("agents.DeleteSession: isolationKey is required")
	}
	if _, err := c.jsonRequest(ctx, http.MethodDelete,
		fmt.Sprintf("%s/agents/%s/endpoint/sessions/%s", c.endpoint, agentName, sessionID),
		nil, "", hostedAndEndpoint,
		map[string]string{"x-session-isolation-key": isolationKey}, http.StatusNoContent); err != nil {
		return DeleteSessionResponse{}, err
	}
	return DeleteSessionResponse{}, nil
}

// ListSessionsOptions is the optional parameter set for NewListSessionsPager.
type ListSessionsOptions struct {
	Limit  *int32
	Order  *PageOrder
	After  *string
	Before *string
}

// NewListSessionsPager returns a Pager for GET /agents/{name}/endpoint/sessions.
func (c *Client) NewListSessionsPager(agentName string, opts *ListSessionsOptions) *runtime.Pager[SessionsPage] {
	first := true
	return runtime.NewPager(runtime.PagingHandler[SessionsPage]{
		More: func(page SessionsPage) bool {
			return page.HasMore && page.LastID != ""
		},
		Fetcher: func(ctx context.Context, page *SessionsPage) (SessionsPage, error) {
			if agentName == "" {
				return SessionsPage{}, errors.New("agents.ListSessions: agentName is required")
			}
			req, err := runtime.NewRequest(ctx, http.MethodGet,
				fmt.Sprintf("%s/agents/%s/endpoint/sessions", c.endpoint, agentName))
			if err != nil {
				return SessionsPage{}, err
			}
			req.Raw().Header.Set("foundry-features", hostedAndEndpoint)
			req.Raw().Header.Set("Accept", "application/json")
			q := req.Raw().URL.Query()
			if opts != nil {
				if opts.Limit != nil {
					q.Set("limit", strconv.FormatInt(int64(*opts.Limit), 10))
				}
				if opts.Order != nil {
					q.Set("order", string(*opts.Order))
				}
				if opts.Before != nil {
					q.Set("before", *opts.Before)
				}
			}
			switch {
			case first:
				first = false
				if opts != nil && opts.After != nil {
					q.Set("after", *opts.After)
				}
			case page != nil:
				q.Set("after", page.LastID)
			}
			q.Set("api-version", c.apiVersion)
			req.Raw().URL.RawQuery = q.Encode()
			body, err := c.do(req, http.StatusOK)
			if err != nil {
				return SessionsPage{}, err
			}
			var out SessionsPage
			if err := json.Unmarshal(body, &out); err != nil {
				return SessionsPage{}, fmt.Errorf("agents.ListSessions: decode: %w", err)
			}
			return out, nil
		},
	})
}

// ListSessionFilesOptions is the optional parameter set for ListSessionFiles.
type ListSessionFilesOptions struct{}

// ListSessionFiles lists files at a path in the session sandbox (non-recursive).
// GET /agents/{name}/endpoint/sessions/{id}/files?path=...
func (c *Client) ListSessionFiles(ctx context.Context, agentName, sessionID, path string, _ *ListSessionFilesOptions) (SessionDirectoryListResponse, error) {
	if agentName == "" {
		return SessionDirectoryListResponse{}, errors.New("agents.ListSessionFiles: agentName is required")
	}
	if sessionID == "" {
		return SessionDirectoryListResponse{}, errors.New("agents.ListSessionFiles: sessionID is required")
	}
	if path == "" {
		return SessionDirectoryListResponse{}, errors.New("agents.ListSessionFiles: path is required")
	}
	req, err := runtime.NewRequest(ctx, http.MethodGet,
		fmt.Sprintf("%s/agents/%s/endpoint/sessions/%s/files", c.endpoint, agentName, sessionID))
	if err != nil {
		return SessionDirectoryListResponse{}, err
	}
	req.Raw().Header.Set("foundry-features", hostedOnly)
	req.Raw().Header.Set("Accept", "application/json")
	q := req.Raw().URL.Query()
	q.Set("path", path)
	q.Set("api-version", c.apiVersion)
	req.Raw().URL.RawQuery = q.Encode()
	body, err := c.do(req, http.StatusOK)
	if err != nil {
		return SessionDirectoryListResponse{}, err
	}
	var out SessionDirectoryListResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return SessionDirectoryListResponse{}, fmt.Errorf("agents.ListSessionFiles: decode: %w", err)
	}
	return out, nil
}

// DeleteSessionFileOptions is the optional parameter set for DeleteSessionFile.
type DeleteSessionFileOptions struct {
	Recursive *bool
}

// DeleteSessionFileResponse is the (empty) response of DeleteSessionFile.
type DeleteSessionFileResponse struct{}

// DeleteSessionFile deletes a file or directory in the session sandbox.
// DELETE .../files?path=...&recursive=... returns 204.
func (c *Client) DeleteSessionFile(ctx context.Context, agentName, sessionID, path string, opts *DeleteSessionFileOptions) (DeleteSessionFileResponse, error) {
	if agentName == "" {
		return DeleteSessionFileResponse{}, errors.New("agents.DeleteSessionFile: agentName is required")
	}
	if sessionID == "" {
		return DeleteSessionFileResponse{}, errors.New("agents.DeleteSessionFile: sessionID is required")
	}
	if path == "" {
		return DeleteSessionFileResponse{}, errors.New("agents.DeleteSessionFile: path is required")
	}
	req, err := runtime.NewRequest(ctx, http.MethodDelete,
		fmt.Sprintf("%s/agents/%s/endpoint/sessions/%s/files", c.endpoint, agentName, sessionID))
	if err != nil {
		return DeleteSessionFileResponse{}, err
	}
	req.Raw().Header.Set("foundry-features", hostedOnly)
	q := req.Raw().URL.Query()
	q.Set("path", path)
	if opts != nil && opts.Recursive != nil {
		q.Set("recursive", strconv.FormatBool(*opts.Recursive))
	}
	q.Set("api-version", c.apiVersion)
	req.Raw().URL.RawQuery = q.Encode()
	if _, err := c.do(req, http.StatusNoContent); err != nil {
		return DeleteSessionFileResponse{}, err
	}
	return DeleteSessionFileResponse{}, nil
}

// DownloadSessionFileOptions is the optional parameter set for DownloadSessionFile.
type DownloadSessionFileOptions struct{}

// DownloadSessionFileResponse wraps the binary file body. Caller must Close.
type DownloadSessionFileResponse struct {
	Body io.ReadCloser
}

// DownloadSessionFile downloads a file from the session sandbox.
// GET .../files/content?path=... returns 200 with application/octet-stream body.
func (c *Client) DownloadSessionFile(ctx context.Context, agentName, sessionID, path string, _ *DownloadSessionFileOptions) (DownloadSessionFileResponse, error) {
	if agentName == "" {
		return DownloadSessionFileResponse{}, errors.New("agents.DownloadSessionFile: agentName is required")
	}
	if sessionID == "" {
		return DownloadSessionFileResponse{}, errors.New("agents.DownloadSessionFile: sessionID is required")
	}
	if path == "" {
		return DownloadSessionFileResponse{}, errors.New("agents.DownloadSessionFile: path is required")
	}
	req, err := runtime.NewRequest(ctx, http.MethodGet,
		fmt.Sprintf("%s/agents/%s/endpoint/sessions/%s/files/content", c.endpoint, agentName, sessionID))
	if err != nil {
		return DownloadSessionFileResponse{}, err
	}
	req.Raw().Header.Set("foundry-features", hostedAndEndpoint)
	req.Raw().Header.Set("Accept", "application/octet-stream")
	q := req.Raw().URL.Query()
	q.Set("path", path)
	q.Set("api-version", c.apiVersion)
	req.Raw().URL.RawQuery = q.Encode()
	resp, err := c.pl.Do(req)
	if err != nil {
		return DownloadSessionFileResponse{}, err
	}
	if !runtime.HasStatusCode(resp, http.StatusOK) {
		return DownloadSessionFileResponse{}, runtime.NewResponseError(resp)
	}
	return DownloadSessionFileResponse{Body: resp.Body}, nil
}

// UploadSessionFileOptions is the optional parameter set for UploadSessionFile.
type UploadSessionFileOptions struct{}

// UploadSessionFile uploads a binary file to the session sandbox.
// PUT .../files/content?path=... accepts 200 or 201.
func (c *Client) UploadSessionFile(ctx context.Context, agentName, sessionID, path string, content []byte, _ *UploadSessionFileOptions) (SessionFileWriteResponse, error) {
	if agentName == "" {
		return SessionFileWriteResponse{}, errors.New("agents.UploadSessionFile: agentName is required")
	}
	if sessionID == "" {
		return SessionFileWriteResponse{}, errors.New("agents.UploadSessionFile: sessionID is required")
	}
	if path == "" {
		return SessionFileWriteResponse{}, errors.New("agents.UploadSessionFile: path is required")
	}
	if len(content) == 0 {
		return SessionFileWriteResponse{}, errors.New("agents.UploadSessionFile: content is required")
	}
	req, err := runtime.NewRequest(ctx, http.MethodPut,
		fmt.Sprintf("%s/agents/%s/endpoint/sessions/%s/files/content", c.endpoint, agentName, sessionID))
	if err != nil {
		return SessionFileWriteResponse{}, err
	}
	req.Raw().Header.Set("foundry-features", hostedAndEndpoint)
	req.Raw().Header.Set("Accept", "application/json")
	q := req.Raw().URL.Query()
	q.Set("path", path)
	q.Set("api-version", c.apiVersion)
	req.Raw().URL.RawQuery = q.Encode()
	if err := req.SetBody(byteSeeker{bytes.NewReader(content)}, "application/octet-stream"); err != nil {
		return SessionFileWriteResponse{}, err
	}
	resp, err := c.pl.Do(req)
	if err != nil {
		return SessionFileWriteResponse{}, err
	}
	if !runtime.HasStatusCode(resp, http.StatusOK, http.StatusCreated) {
		return SessionFileWriteResponse{}, runtime.NewResponseError(resp)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return SessionFileWriteResponse{}, err
	}
	var out SessionFileWriteResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return SessionFileWriteResponse{}, fmt.Errorf("agents.UploadSessionFile: decode: %w", err)
	}
	return out, nil
}

func (c *Client) jsonRequest(ctx context.Context, method, url string, body []byte, contentType, foundryFeat string, extraHeaders map[string]string, okStatuses ...int) ([]byte, error) {
	req, err := runtime.NewRequest(ctx, method, url)
	if err != nil {
		return nil, err
	}
	if foundryFeat == "" {
		foundryFeat = hostedAndEndpoint
	}
	req.Raw().Header.Set("foundry-features", foundryFeat)
	req.Raw().Header.Set("Accept", "application/json")
	for k, v := range extraHeaders {
		req.Raw().Header.Set(k, v)
	}
	q := req.Raw().URL.Query()
	q.Set("api-version", c.apiVersion)
	req.Raw().URL.RawQuery = q.Encode()
	if len(body) > 0 {
		if err := req.SetBody(byteSeeker{bytes.NewReader(body)}, contentType); err != nil {
			return nil, err
		}
	}
	resp, err := c.pl.Do(req)
	if err != nil {
		return nil, err
	}
	if !runtime.HasStatusCode(resp, okStatuses...) {
		return nil, runtime.NewResponseError(resp)
	}
	if resp.StatusCode == http.StatusNoContent || resp.Body == nil {
		return nil, nil
	}
	return io.ReadAll(resp.Body)
}

func (c *Client) do(req *policy.Request, okStatuses ...int) ([]byte, error) {
	resp, err := c.pl.Do(req)
	if err != nil {
		return nil, err
	}
	if !runtime.HasStatusCode(resp, okStatuses...) {
		return nil, runtime.NewResponseError(resp)
	}
	if resp.StatusCode == http.StatusNoContent || resp.Body == nil {
		return nil, nil
	}
	return io.ReadAll(resp.Body)
}

type byteSeeker struct{ *bytes.Reader }

func (byteSeeker) Close() error { return nil }
