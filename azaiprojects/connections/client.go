package connections

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"

	"github.com/sambo/ai-projects-go/azaiprojects/internal/shared"
)

const (
	moduleName    = "azaiprojects/connections"
	moduleVersion = "0.1.0"
	defaultScope  = "https://ai.azure.com/.default"
	defaultAPIVer = "v1"
)

// Client provides the Connections operation group.
type Client struct {
	endpoint   string
	apiVersion string
	pl         runtime.Pipeline
}

// ClientOptions configures the Connections client.
type ClientOptions struct {
	azcore.ClientOptions

	// APIVersion overrides the default service API version.
	APIVersion string
}

// NewClient constructs a Connections client targeting endpoint.
//
// The client takes its own credential (mirrors azure-sdk-for-go convention).
// In typical use, callers reach this through azaiprojects.Client.Connections().
func NewClient(endpoint string, cred azcore.TokenCredential, opts *ClientOptions) (*Client, error) {
	if endpoint == "" {
		return nil, errors.New("connections: endpoint is required")
	}
	if cred == nil {
		return nil, errors.New("connections: cred is required")
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

// NewClientFromPipeline constructs a Connections client that reuses an
// existing pipeline (typically the one from azaiprojects.Client). This is
// the path the parent client uses to avoid building a second pipeline.
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
	// ConnectionType filters the list by connection category.
	ConnectionType *ConnectionType
	// DefaultConnection, when true, only returns connections marked as default.
	DefaultConnection *bool
}

// ListResponse is one page of Connection results.
type ListResponse struct {
	shared.PageResponse[Connection]
}

// NewListPager returns a Pager that issues GET /connections and follows
// nextLink continuations.
func (c *Client) NewListPager(opts *ListOptions) *runtime.Pager[ListResponse] {
	first := true
	return runtime.NewPager(runtime.PagingHandler[ListResponse]{
		More: func(page ListResponse) bool {
			return page.NextLink != nil && *page.NextLink != ""
		},
		Fetcher: func(ctx context.Context, page *ListResponse) (ListResponse, error) {
			var url string
			if first || page == nil || page.NextLink == nil {
				first = false
				url = fmt.Sprintf("%s/connections", c.endpoint)
			} else {
				url = *page.NextLink
			}
			req, err := runtime.NewRequest(ctx, http.MethodGet, url)
			if err != nil {
				return ListResponse{}, err
			}
			q := req.Raw().URL.Query()
			q.Set("api-version", c.apiVersion)
			if opts != nil {
				if opts.ConnectionType != nil {
					q.Set("connectionType", string(*opts.ConnectionType))
				}
				if opts.DefaultConnection != nil {
					if *opts.DefaultConnection {
						q.Set("defaultConnection", "true")
					} else {
						q.Set("defaultConnection", "false")
					}
				}
			}
			req.Raw().URL.RawQuery = q.Encode()

			resp, err := c.pl.Do(req)
			if err != nil {
				return ListResponse{}, err
			}
			if !runtime.HasStatusCode(resp, http.StatusOK) {
				return ListResponse{}, runtime.NewResponseError(resp)
			}
			var out ListResponse
			if err := runtime.UnmarshalAsJSON(resp, &out); err != nil {
				return ListResponse{}, err
			}
			return out, nil
		},
	})
}
