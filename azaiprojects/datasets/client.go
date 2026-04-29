package datasets

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
	moduleName    = "azaiprojects/datasets"
	moduleVersion = "0.1.0"
	defaultScope  = "https://ai.azure.com/.default"
	defaultAPIVer = "v1"
)

// Client provides the Datasets operation group.
type Client struct {
	endpoint   string
	apiVersion string
	pl         runtime.Pipeline
}

// ClientOptions configures the Datasets client.
type ClientOptions struct {
	azcore.ClientOptions
	APIVersion string
}

// NewClient constructs a Datasets client targeting endpoint.
func NewClient(endpoint string, cred azcore.TokenCredential, opts *ClientOptions) (*Client, error) {
	if endpoint == "" {
		return nil, errors.New("datasets: endpoint is required")
	}
	if cred == nil {
		return nil, errors.New("datasets: cred is required")
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

// DatasetValue is the element type returned by list endpoints. It carries
// every DatasetVersion field plus the raw JSON for type-specific decoding.
type DatasetValue struct {
	DatasetVersion
	Raw json.RawMessage `json:"-"`
}

// UnmarshalJSON keeps the original JSON in Raw for re-decoding.
func (v *DatasetValue) UnmarshalJSON(data []byte) error {
	v.Raw = append(v.Raw[:0], data...)
	type alias DatasetVersion
	var a alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	v.DatasetVersion = DatasetVersion(a)
	return nil
}

// ListOptions is the optional parameter set for NewListPager.
type ListOptions struct{}

// ListResponse is one page of DatasetValue results.
type ListResponse struct {
	shared.PageResponse[DatasetValue]
}

// NewListPager returns a Pager that issues GET /datasets.
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
				url = fmt.Sprintf("%s/datasets", c.endpoint)
			} else {
				url = *page.NextLink
			}
			return c.fetchPage(ctx, url)
		},
	})
}

// ListVersionsOptions is the optional parameter set for NewListVersionsPager.
type ListVersionsOptions struct{}

// NewListVersionsPager returns a Pager that issues GET /datasets/{name}/versions.
func (c *Client) NewListVersionsPager(name string, _ *ListVersionsOptions) *runtime.Pager[ListResponse] {
	first := true
	return runtime.NewPager(runtime.PagingHandler[ListResponse]{
		More: func(page ListResponse) bool {
			return page.NextLink != nil && *page.NextLink != ""
		},
		Fetcher: func(ctx context.Context, page *ListResponse) (ListResponse, error) {
			if name == "" {
				return ListResponse{}, errors.New("datasets.NewListVersionsPager: name is required")
			}
			var url string
			if first || page == nil || page.NextLink == nil {
				first = false
				url = fmt.Sprintf("%s/datasets/%s/versions", c.endpoint, name)
			} else {
				url = *page.NextLink
			}
			return c.fetchPage(ctx, url)
		},
	})
}

func (c *Client) fetchPage(ctx context.Context, url string) (ListResponse, error) {
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
}

// GetOptions is the optional parameter set for Get.
type GetOptions struct{}

// GetResponse wraps a single DatasetValue with the raw JSON for typed decoding.
type GetResponse struct {
	DatasetValue
}

// Get retrieves the specific version of a DatasetVersion.
func (c *Client) Get(ctx context.Context, name, version string, _ *GetOptions) (GetResponse, error) {
	if name == "" || version == "" {
		return GetResponse{}, errors.New("datasets.Get: name and version are required")
	}
	url := fmt.Sprintf("%s/datasets/%s/versions/%s", c.endpoint, name, version)
	body, err := c.simpleJSONRequest(ctx, http.MethodGet, url, nil, "", http.StatusOK)
	if err != nil {
		return GetResponse{}, err
	}
	var out GetResponse
	if err := json.Unmarshal(body, &out.DatasetValue); err != nil {
		return GetResponse{}, err
	}
	return out, nil
}

// CreateOrUpdateOptions is the optional parameter set for CreateOrUpdate.
type CreateOrUpdateOptions struct{}

// CreateOrUpdateResponse wraps the resulting DatasetValue.
type CreateOrUpdateResponse struct {
	DatasetValue
}

// CreateOrUpdate creates a new or updates an existing DatasetVersion.
//
// Body must JSON-marshal to a valid DatasetVersionUnion (FileDatasetVersion,
// FolderDatasetVersion, or a bare DatasetVersion).
func (c *Client) CreateOrUpdate(ctx context.Context, name, version string, body any, _ *CreateOrUpdateOptions) (CreateOrUpdateResponse, error) {
	if name == "" || version == "" {
		return CreateOrUpdateResponse{}, errors.New("datasets.CreateOrUpdate: name and version are required")
	}
	if body == nil {
		return CreateOrUpdateResponse{}, errors.New("datasets.CreateOrUpdate: body is required")
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return CreateOrUpdateResponse{}, fmt.Errorf("datasets.CreateOrUpdate: marshal: %w", err)
	}
	url := fmt.Sprintf("%s/datasets/%s/versions/%s", c.endpoint, name, version)
	respBody, err := c.simpleJSONRequest(ctx, http.MethodPatch, url, payload, "application/merge-patch+json", http.StatusOK, http.StatusCreated)
	if err != nil {
		return CreateOrUpdateResponse{}, err
	}
	var out CreateOrUpdateResponse
	if err := json.Unmarshal(respBody, &out.DatasetValue); err != nil {
		return CreateOrUpdateResponse{}, err
	}
	return out, nil
}

// DeleteOptions is the optional parameter set for Delete.
type DeleteOptions struct{}

// DeleteResponse is empty; Delete is success/failure.
type DeleteResponse struct{}

// Delete removes a specific DatasetVersion.
func (c *Client) Delete(ctx context.Context, name, version string, _ *DeleteOptions) (DeleteResponse, error) {
	if name == "" || version == "" {
		return DeleteResponse{}, errors.New("datasets.Delete: name and version are required")
	}
	url := fmt.Sprintf("%s/datasets/%s/versions/%s", c.endpoint, name, version)
	if _, err := c.simpleJSONRequest(ctx, http.MethodDelete, url, nil, "", http.StatusNoContent, http.StatusOK); err != nil {
		return DeleteResponse{}, err
	}
	return DeleteResponse{}, nil
}

// PendingUploadOptions is the optional parameter set for PendingUpload.
type PendingUploadOptions struct{}

// PendingUpload starts (or resumes) a pending upload for a dataset version.
//
// POST /datasets/{name}/versions/{version}/startPendingUpload?api-version=v1.
func (c *Client) PendingUpload(ctx context.Context, name, version string, req PendingUploadRequest, _ *PendingUploadOptions) (PendingUploadResponse, error) {
	if name == "" || version == "" {
		return PendingUploadResponse{}, errors.New("datasets.PendingUpload: name and version are required")
	}
	payload, err := json.Marshal(req)
	if err != nil {
		return PendingUploadResponse{}, fmt.Errorf("datasets.PendingUpload: marshal: %w", err)
	}
	url := fmt.Sprintf("%s/datasets/%s/versions/%s/startPendingUpload", c.endpoint, name, version)
	respBody, err := c.simpleJSONRequest(ctx, http.MethodPost, url, payload, "application/json", http.StatusOK)
	if err != nil {
		return PendingUploadResponse{}, err
	}
	var out PendingUploadResponse
	if err := json.Unmarshal(respBody, &out); err != nil {
		return PendingUploadResponse{}, err
	}
	return out, nil
}

// GetCredentialsOptions is the optional parameter set for GetCredentials.
type GetCredentialsOptions struct{}

// GetCredentials returns the SAS credential for accessing a dataset version's storage.
//
// POST /datasets/{name}/versions/{version}/credentials?api-version=v1.
func (c *Client) GetCredentials(ctx context.Context, name, version string, _ *GetCredentialsOptions) (DatasetCredential, error) {
	if name == "" || version == "" {
		return DatasetCredential{}, errors.New("datasets.GetCredentials: name and version are required")
	}
	url := fmt.Sprintf("%s/datasets/%s/versions/%s/credentials", c.endpoint, name, version)
	respBody, err := c.simpleJSONRequest(ctx, http.MethodPost, url, []byte("{}"), "application/json", http.StatusOK)
	if err != nil {
		return DatasetCredential{}, err
	}
	var out DatasetCredential
	if err := json.Unmarshal(respBody, &out); err != nil {
		return DatasetCredential{}, err
	}
	return out, nil
}

// simpleJSONRequest issues a request to url, applying api-version, optional
// JSON body, and validating the response status against okStatuses. Returns
// the response body bytes (empty for 204).
func (c *Client) simpleJSONRequest(ctx context.Context, method, url string, body []byte, contentType string, okStatuses ...int) ([]byte, error) {
	req, err := runtime.NewRequest(ctx, method, url)
	if err != nil {
		return nil, err
	}
	q := req.Raw().URL.Query()
	q.Set("api-version", c.apiVersion)
	req.Raw().URL.RawQuery = q.Encode()
	if len(body) > 0 {
		req.Raw().Header.Set("Accept", "application/json")
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

type byteSeeker struct{ *bytes.Reader }

func (byteSeeker) Close() error { return nil }
