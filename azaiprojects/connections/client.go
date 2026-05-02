package connections

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"

	"github.com/retryrecv/azure-ai-projects-go/azaiprojects/internal/shared"
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

// newGetRequest builds a GET request to {endpoint}{path} with api-version
// applied to the query string.
func (c *Client) newGetRequest(ctx context.Context, path string) (*policy.Request, error) {
	req, err := runtime.NewRequest(ctx, http.MethodGet, c.endpoint+path)
	if err != nil {
		return nil, err
	}
	q := req.Raw().URL.Query()
	q.Set("api-version", c.apiVersion)
	req.Raw().URL.RawQuery = q.Encode()
	return req, nil
}

// GetWithCredentialsOptions is the optional parameter set for GetWithCredentials.
type GetWithCredentialsOptions struct{}

// GetDefaultOptions is the optional parameter set for GetDefault.
type GetDefaultOptions struct {
	// IncludeCredentials, when true, follows up the list call with a
	// GetWithCredentials on the returned connection. Mirrors the TypeScript
	// sample at samples-dev/connections/connectionsBasics.ts.
	IncludeCredentials bool
}

// GetDefaultResponse wraps the default Connection for a type.
type GetDefaultResponse struct {
	Connection
}

// GetDefault returns the default Connection of a given type.
//
// Implementation matches @azure/ai-projects: list connections filtered to
// the requested type with defaultConnection=true, return the first hit.
// When opts.IncludeCredentials is true, a follow-up GetWithCredentials
// call populates the credentials.
func (c *Client) GetDefault(ctx context.Context, connectionType ConnectionType, opts *GetDefaultOptions) (GetDefaultResponse, error) {
	if connectionType == "" {
		return GetDefaultResponse{}, errors.New("connections.GetDefault: connectionType is required")
	}
	defaultTrue := true
	pager := c.NewListPager(&ListOptions{
		ConnectionType:    &connectionType,
		DefaultConnection: &defaultTrue,
	})
	if !pager.More() {
		return GetDefaultResponse{}, errors.New("connections.GetDefault: pager has no pages")
	}
	page, err := pager.NextPage(ctx)
	if err != nil {
		return GetDefaultResponse{}, err
	}
	if len(page.Value) == 0 {
		return GetDefaultResponse{}, fmt.Errorf("connections.GetDefault: no default connection of type %q", connectionType)
	}
	first := page.Value[0]
	if opts != nil && opts.IncludeCredentials {
		withCreds, err := c.GetWithCredentials(ctx, first.Name, nil)
		if err != nil {
			return GetDefaultResponse{}, err
		}
		return GetDefaultResponse{Connection: withCreds.Connection}, nil
	}
	return GetDefaultResponse{Connection: first}, nil
}

// GetOptions is the optional parameter set for Get.
type GetOptions struct{}

// GetResponse wraps a single Connection.
type GetResponse struct {
	Connection
}

// Get retrieves a single connection by name (without credentials).
//
// Mirrors GET /connections/{name}?api-version=v1.
func (c *Client) Get(ctx context.Context, name string, _ *GetOptions) (GetResponse, error) {
	if name == "" {
		return GetResponse{}, errors.New("connections.Get: name is required")
	}
	req, err := c.newGetRequest(ctx, "/connections/"+name)
	if err != nil {
		return GetResponse{}, err
	}
	resp, err := c.pl.Do(req)
	if err != nil {
		return GetResponse{}, err
	}
	if !runtime.HasStatusCode(resp, http.StatusOK) {
		return GetResponse{}, runtime.NewResponseError(resp)
	}
	var out GetResponse
	if err := runtime.UnmarshalAsJSON(resp, &out.Connection); err != nil {
		return GetResponse{}, err
	}
	return out, nil
}

// GetWithCredentialsResponse wraps a single Connection populated with credentials.
type GetWithCredentialsResponse struct {
	Connection
}

// GetWithCredentials retrieves a single connection by name, including credentials.
//
// Mirrors POST /connections/{name}/getConnectionWithCredentials?api-version=v1.
// (The endpoint takes POST, not GET — verified against the live service.)
func (c *Client) GetWithCredentials(ctx context.Context, name string, _ *GetWithCredentialsOptions) (GetWithCredentialsResponse, error) {
	if name == "" {
		return GetWithCredentialsResponse{}, errors.New("connections.GetWithCredentials: name is required")
	}
	url := c.endpoint + "/connections/" + name + "/getConnectionWithCredentials"
	req, err := runtime.NewRequest(ctx, http.MethodPost, url)
	if err != nil {
		return GetWithCredentialsResponse{}, err
	}
	q := req.Raw().URL.Query()
	q.Set("api-version", c.apiVersion)
	req.Raw().URL.RawQuery = q.Encode()
	req.Raw().Header.Set("Accept", "application/json")

	resp, err := c.pl.Do(req)
	if err != nil {
		return GetWithCredentialsResponse{}, err
	}
	if !runtime.HasStatusCode(resp, http.StatusOK) {
		return GetWithCredentialsResponse{}, runtime.NewResponseError(resp)
	}
	var out GetWithCredentialsResponse
	if err := runtime.UnmarshalAsJSON(resp, &out.Connection); err != nil {
		return GetWithCredentialsResponse{}, err
	}
	return out, nil
}

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
