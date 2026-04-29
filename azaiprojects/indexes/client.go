package indexes

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

	"github.com/sambo/ai-projects-go/azaiprojects/internal/shared"
)

const (
	moduleName    = "azaiprojects/indexes"
	moduleVersion = "0.1.0"
	defaultScope  = "https://ai.azure.com/.default"
	defaultAPIVer = "v1"
)

// Client provides the Indexes operation group.
type Client struct {
	endpoint   string
	apiVersion string
	pl         runtime.Pipeline
}

// ClientOptions configures the Indexes client.
type ClientOptions struct {
	azcore.ClientOptions
	APIVersion string
}

// NewClient constructs an Indexes client targeting endpoint.
func NewClient(endpoint string, cred azcore.TokenCredential, opts *ClientOptions) (*Client, error) {
	if endpoint == "" {
		return nil, errors.New("indexes: endpoint is required")
	}
	if cred == nil {
		return nil, errors.New("indexes: cred is required")
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
type ListOptions struct{}

// ListResponse is one page of Index results.
//
// The wire returns the IndexUnion, but for now the Go shape collapses the
// union onto a generic map so callers can read fields by name. A typed
// discriminated result lands in a follow-up task.
type ListResponse struct {
	shared.PageResponse[IndexValue]
}

// IndexValue is the element type returned by list endpoints. It carries
// every Index field plus the type-specific fields as raw JSON the caller
// can decode into AzureAISearchIndex / ManagedAzureAISearchIndex / CosmosDBIndex.
type IndexValue struct {
	Index
	Raw json.RawMessage `json:"-"`
}

// UnmarshalJSON keeps the original JSON in Raw so callers can re-decode
// into a concrete type when needed.
func (v *IndexValue) UnmarshalJSON(data []byte) error {
	v.Raw = append(v.Raw[:0], data...)
	type alias Index
	var a alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	v.Index = Index(a)
	return nil
}

// NewListPager returns a Pager that issues GET /indexes.
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
				url = fmt.Sprintf("%s/indexes", c.endpoint)
			} else {
				url = *page.NextLink
			}
			req, err := runtime.NewRequest(ctx, http.MethodGet, url)
			if err != nil {
				return ListResponse{}, err
			}
			q := req.Raw().URL.Query()
			q.Set("api-version", c.apiVersion)
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

// ListVersionsOptions is the optional parameter set for NewListVersionsPager.
type ListVersionsOptions struct{}

// NewListVersionsPager returns a Pager that issues GET /indexes/{name}/versions.
func (c *Client) NewListVersionsPager(name string, _ *ListVersionsOptions) *runtime.Pager[ListResponse] {
	first := true
	return runtime.NewPager(runtime.PagingHandler[ListResponse]{
		More: func(page ListResponse) bool {
			return page.NextLink != nil && *page.NextLink != ""
		},
		Fetcher: func(ctx context.Context, page *ListResponse) (ListResponse, error) {
			if name == "" {
				return ListResponse{}, errors.New("indexes.NewListVersionsPager: name is required")
			}
			var url string
			if first || page == nil || page.NextLink == nil {
				first = false
				url = fmt.Sprintf("%s/indexes/%s/versions", c.endpoint, name)
			} else {
				url = *page.NextLink
			}
			req, err := runtime.NewRequest(ctx, http.MethodGet, url)
			if err != nil {
				return ListResponse{}, err
			}
			q := req.Raw().URL.Query()
			q.Set("api-version", c.apiVersion)
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

// GetResponse wraps a single Index value with its raw JSON for type-specific decoding.
type GetResponse struct {
	IndexValue
}

// Get retrieves a specific version of an index.
func (c *Client) Get(ctx context.Context, name, version string, _ *GetOptions) (GetResponse, error) {
	if name == "" || version == "" {
		return GetResponse{}, errors.New("indexes.Get: name and version are required")
	}
	url := fmt.Sprintf("%s/indexes/%s/versions/%s", c.endpoint, name, version)
	req, err := runtime.NewRequest(ctx, http.MethodGet, url)
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
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return GetResponse{}, err
	}
	var out GetResponse
	if err := json.Unmarshal(body, &out.IndexValue); err != nil {
		return GetResponse{}, err
	}
	return out, nil
}

// CreateOrUpdateOptions is the optional parameter set for CreateOrUpdate.
type CreateOrUpdateOptions struct{}

// CreateOrUpdateResponse wraps the resulting Index value.
type CreateOrUpdateResponse struct {
	IndexValue
}

// CreateOrUpdate creates a new or updates an existing Index version.
//
// The body parameter must be a value (or pointer to a value) that
// JSON-marshals to a valid IndexUnion: AzureAISearchIndex,
// ManagedAzureAISearchIndex, CosmosDBIndex, or a bare Index.
func (c *Client) CreateOrUpdate(ctx context.Context, name, version string, body any, _ *CreateOrUpdateOptions) (CreateOrUpdateResponse, error) {
	if name == "" || version == "" {
		return CreateOrUpdateResponse{}, errors.New("indexes.CreateOrUpdate: name and version are required")
	}
	if body == nil {
		return CreateOrUpdateResponse{}, errors.New("indexes.CreateOrUpdate: body is required")
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return CreateOrUpdateResponse{}, fmt.Errorf("indexes.CreateOrUpdate: marshal: %w", err)
	}

	url := fmt.Sprintf("%s/indexes/%s/versions/%s", c.endpoint, name, version)
	req, err := runtime.NewRequest(ctx, http.MethodPatch, url)
	if err != nil {
		return CreateOrUpdateResponse{}, err
	}
	q := req.Raw().URL.Query()
	q.Set("api-version", c.apiVersion)
	req.Raw().URL.RawQuery = q.Encode()
	req.Raw().Header.Set("Accept", "application/json")
	if err := req.SetBody(streamSeeker(payload), "application/merge-patch+json"); err != nil {
		return CreateOrUpdateResponse{}, err
	}

	resp, err := c.pl.Do(req)
	if err != nil {
		return CreateOrUpdateResponse{}, err
	}
	if !runtime.HasStatusCode(resp, http.StatusOK, http.StatusCreated) {
		return CreateOrUpdateResponse{}, runtime.NewResponseError(resp)
	}
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return CreateOrUpdateResponse{}, err
	}
	var out CreateOrUpdateResponse
	if err := json.Unmarshal(respBody, &out.IndexValue); err != nil {
		return CreateOrUpdateResponse{}, err
	}
	return out, nil
}

// DeleteOptions is the optional parameter set for Delete.
type DeleteOptions struct{}

// DeleteResponse is empty; Delete is success/failure.
type DeleteResponse struct{}

// Delete removes the specific version of an Index. 204 and 200 are both
// treated as success.
func (c *Client) Delete(ctx context.Context, name, version string, _ *DeleteOptions) (DeleteResponse, error) {
	if name == "" || version == "" {
		return DeleteResponse{}, errors.New("indexes.Delete: name and version are required")
	}
	url := fmt.Sprintf("%s/indexes/%s/versions/%s", c.endpoint, name, version)
	req, err := runtime.NewRequest(ctx, http.MethodDelete, url)
	if err != nil {
		return DeleteResponse{}, err
	}
	q := req.Raw().URL.Query()
	q.Set("api-version", c.apiVersion)
	req.Raw().URL.RawQuery = q.Encode()

	resp, err := c.pl.Do(req)
	if err != nil {
		return DeleteResponse{}, err
	}
	if !runtime.HasStatusCode(resp, http.StatusNoContent, http.StatusOK) {
		return DeleteResponse{}, runtime.NewResponseError(resp)
	}
	return DeleteResponse{}, nil
}

// streamSeeker adapts a byte slice to io.ReadSeekCloser for SetBody.
type byteSeeker struct {
	*bytes.Reader
}

func (byteSeeker) Close() error { return nil }

func streamSeeker(b []byte) byteSeeker { return byteSeeker{bytes.NewReader(b)} }
