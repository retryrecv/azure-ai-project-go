package deployments

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
	moduleName    = "azaiprojects/deployments"
	moduleVersion = "0.1.0"
	defaultScope  = "https://ai.azure.com/.default"
	defaultAPIVer = "v1"
)

// Client provides the Deployments operation group.
type Client struct {
	endpoint   string
	apiVersion string
	pl         runtime.Pipeline
}

// ClientOptions configures the Deployments client.
type ClientOptions struct {
	azcore.ClientOptions
	APIVersion string
}

// NewClient constructs a Deployments client targeting endpoint.
func NewClient(endpoint string, cred azcore.TokenCredential, opts *ClientOptions) (*Client, error) {
	if endpoint == "" {
		return nil, errors.New("deployments: endpoint is required")
	}
	if cred == nil {
		return nil, errors.New("deployments: cred is required")
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

// NewClientFromPipeline constructs a Deployments client that reuses an
// existing pipeline (typically from azaiprojects.Client).
func NewClientFromPipeline(endpoint, apiVersion string, pl runtime.Pipeline) *Client {
	if apiVersion == "" {
		apiVersion = defaultAPIVer
	}
	return &Client{endpoint: endpoint, apiVersion: apiVersion, pl: pl}
}

// Endpoint returns the configured service endpoint.
func (c *Client) Endpoint() string { return c.endpoint }

// ListOptions filters the deployments list.
type ListOptions struct {
	ModelPublisher *string
	ModelName      *string
	DeploymentType *DeploymentType
}

// ListResponse is one page of ModelDeployment results.
type ListResponse struct {
	shared.PageResponse[ModelDeployment]
}

// NewListPager returns a Pager that issues GET /deployments.
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
				url = fmt.Sprintf("%s/deployments", c.endpoint)
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
				if opts.ModelPublisher != nil {
					q.Set("modelPublisher", *opts.ModelPublisher)
				}
				if opts.ModelName != nil {
					q.Set("modelName", *opts.ModelName)
				}
				if opts.DeploymentType != nil {
					q.Set("deploymentType", string(*opts.DeploymentType))
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

// GetOptions is the optional parameter set for Get.
type GetOptions struct{}

// GetResponse wraps a single ModelDeployment.
type GetResponse struct {
	ModelDeployment
}

// Get retrieves a deployment by name (GET /deployments/{name}?api-version=v1).
func (c *Client) Get(ctx context.Context, name string, _ *GetOptions) (GetResponse, error) {
	if name == "" {
		return GetResponse{}, errors.New("deployments.Get: name is required")
	}
	req, err := runtime.NewRequest(ctx, http.MethodGet, c.endpoint+"/deployments/"+name)
	if err != nil {
		return GetResponse{}, err
	}
	q := req.Raw().URL.Query()
	q.Set("api-version", c.apiVersion)
	req.Raw().URL.RawQuery = q.Encode()

	resp, err := c.pl.Do(req)
	if err != nil {
		return GetResponse{}, err
	}
	if !runtime.HasStatusCode(resp, http.StatusOK) {
		return GetResponse{}, runtime.NewResponseError(resp)
	}
	var out GetResponse
	if err := runtime.UnmarshalAsJSON(resp, &out.ModelDeployment); err != nil {
		return GetResponse{}, err
	}
	return out, nil
}
