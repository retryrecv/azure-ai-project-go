package redteams

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"

	"github.com/retryrecv/azure-ai-projects-go/azaiprojects/internal/shared"
)

const (
	moduleName    = "azaiprojects/beta/redteams"
	moduleVersion = "0.1.0"
	defaultScope  = "https://ai.azure.com/.default"
	defaultAPIVer = "v1"
	foundryHeader = "RedTeams=V1Preview"
)

// Client provides the beta.redTeams operation group.
type Client struct {
	endpoint   string
	apiVersion string
	pl         runtime.Pipeline
}

// ClientOptions configures the redteams client.
type ClientOptions struct {
	azcore.ClientOptions
	APIVersion string
}

// NewClient constructs a redteams client targeting endpoint.
func NewClient(endpoint string, cred azcore.TokenCredential, opts *ClientOptions) (*Client, error) {
	if endpoint == "" {
		return nil, errors.New("redteams: endpoint is required")
	}
	if cred == nil {
		return nil, errors.New("redteams: cred is required")
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

// NewClientFromPipeline reuses an existing pipeline (typically from azaiprojects.Client).
func NewClientFromPipeline(endpoint, apiVersion string, pl runtime.Pipeline) *Client {
	if apiVersion == "" {
		apiVersion = defaultAPIVer
	}
	return &Client{endpoint: endpoint, apiVersion: apiVersion, pl: pl}
}

// Endpoint returns the configured service endpoint.
func (c *Client) Endpoint() string { return c.endpoint }

// ListResponse is one page of RedTeam results.
type ListResponse struct {
	shared.PageResponse[RedTeam]
}

// ListOptions is the optional parameter set for NewListPager.
type ListOptions struct{}

// NewListPager returns a Pager that issues GET /redTeams/runs.
func (c *Client) NewListPager(_ *ListOptions) *runtime.Pager[ListResponse] {
	first := true
	return runtime.NewPager(runtime.PagingHandler[ListResponse]{
		More: func(page ListResponse) bool {
			return page.NextLink != nil && *page.NextLink != ""
		},
		Fetcher: func(ctx context.Context, page *ListResponse) (ListResponse, error) {
			var url string
			if first || page == nil || page.NextLink == nil {
				first = false
				url = c.endpoint + "/redTeams/runs"
			} else {
				url = *page.NextLink
			}
			req, err := runtime.NewRequest(ctx, http.MethodGet, url)
			if err != nil {
				return ListResponse{}, err
			}
			req.Raw().Header.Set("foundry-features", foundryHeader)
			req.Raw().Header.Set("Accept", "application/json")
			q := req.Raw().URL.Query()
			q.Set("api-version", c.apiVersion)
			req.Raw().URL.RawQuery = q.Encode()
			body, err := c.do(req, http.StatusOK)
			if err != nil {
				return ListResponse{}, err
			}
			var out ListResponse
			if err := json.Unmarshal(body, &out); err != nil {
				return ListResponse{}, fmt.Errorf("redteams.List: decode: %w", err)
			}
			return out, nil
		},
	})
}

// GetOptions is the optional parameter set for Get.
type GetOptions struct{}

// Get retrieves a redteam by name.
func (c *Client) Get(ctx context.Context, name string, _ *GetOptions) (RedTeam, error) {
	if name == "" {
		return RedTeam{}, errors.New("redteams.Get: name is required")
	}
	body, err := c.jsonRequest(ctx, http.MethodGet,
		fmt.Sprintf("%s/redTeams/runs/%s", c.endpoint, name), nil, "", http.StatusOK)
	if err != nil {
		return RedTeam{}, err
	}
	var out RedTeam
	if err := json.Unmarshal(body, &out); err != nil {
		return RedTeam{}, fmt.Errorf("redteams.Get: decode: %w", err)
	}
	return out, nil
}

// CreateOptions is the optional parameter set for Create.
type CreateOptions struct{}

// Create starts a redteam run.
// POST /redTeams/runs:run returns 201.
func (c *Client) Create(ctx context.Context, redTeam RedTeam, _ *CreateOptions) (RedTeam, error) {
	payload, err := json.Marshal(redTeam)
	if err != nil {
		return RedTeam{}, fmt.Errorf("redteams.Create: marshal: %w", err)
	}
	body, err := c.jsonRequest(ctx, http.MethodPost,
		c.endpoint+"/redTeams/runs:run", payload, "application/json", http.StatusCreated)
	if err != nil {
		return RedTeam{}, err
	}
	var out RedTeam
	if err := json.Unmarshal(body, &out); err != nil {
		return RedTeam{}, fmt.Errorf("redteams.Create: decode: %w", err)
	}
	return out, nil
}

func (c *Client) jsonRequest(ctx context.Context, method, url string, body []byte, contentType string, okStatuses ...int) ([]byte, error) {
	req, err := runtime.NewRequest(ctx, method, url)
	if err != nil {
		return nil, err
	}
	req.Raw().Header.Set("foundry-features", foundryHeader)
	req.Raw().Header.Set("Accept", "application/json")
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
	if resp.Body == nil {
		return nil, nil
	}
	return io.ReadAll(resp.Body)
}

type byteSeeker struct{ *bytes.Reader }

func (byteSeeker) Close() error { return nil }
