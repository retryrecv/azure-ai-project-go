package agents

import (
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
	moduleName    = "azaiprojects/agents"
	moduleVersion = "0.1.0"
	defaultScope  = "https://ai.azure.com/.default"
	defaultAPIVer = "v1"
)

// Client provides the Agents operation group.
type Client struct {
	endpoint   string
	apiVersion string
	pl         runtime.Pipeline
}

// ClientOptions configures the Agents client.
type ClientOptions struct {
	azcore.ClientOptions
	APIVersion string
}

// NewClient constructs an Agents client targeting endpoint.
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

// NewClientFromPipeline reuses an existing pipeline (typically from azaiprojects.Client).
func NewClientFromPipeline(endpoint, apiVersion string, pl runtime.Pipeline) *Client {
	if apiVersion == "" {
		apiVersion = defaultAPIVer
	}
	return &Client{endpoint: endpoint, apiVersion: apiVersion, pl: pl}
}

// Endpoint returns the configured service endpoint.
func (c *Client) Endpoint() string { return c.endpoint }

// ListOptions is the optional parameter set for NewListPager.
type ListOptions struct {
	Kind   *AgentKind
	Limit  *int32
	Order  *PageOrder
	After  *string
	Before *string
}

// NewListPager returns a Pager that issues GET /agents.
//
// Pagination is cursor-based: the service returns has_more + last_id; when
// has_more is true the pager re-issues the same query with after=<last_id>.
// Caller-supplied opts.After is honored on the first page only.
func (c *Client) NewListPager(opts *ListOptions) *runtime.Pager[AgentsPage] {
	first := true
	return runtime.NewPager(runtime.PagingHandler[AgentsPage]{
		More: func(page AgentsPage) bool {
			return page.HasMore && page.LastID != ""
		},
		Fetcher: func(ctx context.Context, page *AgentsPage) (AgentsPage, error) {
			req, err := c.newGetRequest(ctx, "/agents")
			if err != nil {
				return AgentsPage{}, err
			}
			q := req.Raw().URL.Query()
			if opts != nil {
				if opts.Kind != nil {
					q.Set("kind", string(*opts.Kind))
				}
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
			return doJSON[AgentsPage](c, req)
		},
	})
}

// ListVersionsOptions is the optional parameter set for NewListVersionsPager.
type ListVersionsOptions struct {
	Limit  *int32
	Order  *PageOrder
	After  *string
	Before *string
}

// NewListVersionsPager returns a Pager that issues GET /agents/{name}/versions.
func (c *Client) NewListVersionsPager(agentName string, opts *ListVersionsOptions) *runtime.Pager[AgentVersionsPage] {
	first := true
	return runtime.NewPager(runtime.PagingHandler[AgentVersionsPage]{
		More: func(page AgentVersionsPage) bool {
			return page.HasMore && page.LastID != ""
		},
		Fetcher: func(ctx context.Context, page *AgentVersionsPage) (AgentVersionsPage, error) {
			if agentName == "" {
				return AgentVersionsPage{}, errors.New("agents.NewListVersionsPager: agentName is required")
			}
			req, err := c.newGetRequest(ctx, fmt.Sprintf("/agents/%s/versions", agentName))
			if err != nil {
				return AgentVersionsPage{}, err
			}
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
			return doJSON[AgentVersionsPage](c, req)
		},
	})
}

// GetOptions is the optional parameter set for Get.
type GetOptions struct{}

// Get retrieves an agent by name.
func (c *Client) Get(ctx context.Context, agentName string, _ *GetOptions) (Agent, error) {
	if agentName == "" {
		return Agent{}, errors.New("agents.Get: agentName is required")
	}
	req, err := c.newGetRequest(ctx, fmt.Sprintf("/agents/%s", agentName))
	if err != nil {
		return Agent{}, err
	}
	c.setAPIVersion(req)
	return doJSON[Agent](c, req)
}

// GetVersionOptions is the optional parameter set for GetVersion.
type GetVersionOptions struct{}

// GetVersion retrieves a specific version of an agent.
func (c *Client) GetVersion(ctx context.Context, agentName, agentVersion string, _ *GetVersionOptions) (AgentVersion, error) {
	if agentName == "" || agentVersion == "" {
		return AgentVersion{}, errors.New("agents.GetVersion: agentName and agentVersion are required")
	}
	req, err := c.newGetRequest(ctx, fmt.Sprintf("/agents/%s/versions/%s", agentName, agentVersion))
	if err != nil {
		return AgentVersion{}, err
	}
	c.setAPIVersion(req)
	return doJSON[AgentVersion](c, req)
}

// --- shared helpers ---

func (c *Client) newGetRequest(ctx context.Context, path string) (*policy.Request, error) {
	return runtime.NewRequest(ctx, http.MethodGet, c.endpoint+path)
}

// setAPIVersion adds the api-version query param and returns a new RawQuery.
func (c *Client) setAPIVersion(req *policy.Request) {
	q := req.Raw().URL.Query()
	q.Set("api-version", c.apiVersion)
	req.Raw().URL.RawQuery = q.Encode()
}

// doJSON sends req, asserts status 200, and decodes the body into T.
func doJSON[T any](c *Client, req *policy.Request) (T, error) {
	var zero T
	resp, err := c.pl.Do(req)
	if err != nil {
		return zero, err
	}
	if !runtime.HasStatusCode(resp, http.StatusOK) {
		return zero, runtime.NewResponseError(resp)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return zero, fmt.Errorf("agents: read body: %w", err)
	}
	var out T
	if err := json.Unmarshal(body, &out); err != nil {
		return zero, fmt.Errorf("agents: decode body: %w", err)
	}
	return out, nil
}
