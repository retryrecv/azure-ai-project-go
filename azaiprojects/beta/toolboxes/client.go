package toolboxes

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
	moduleName    = "azaiprojects/beta/toolboxes"
	moduleVersion = "0.1.0"
	defaultScope  = "https://ai.azure.com/.default"
	defaultAPIVer = "v1"
	foundryHeader = "Toolboxes=V1Preview"
)

// Client provides the beta.toolboxes operation group.
type Client struct {
	endpoint   string
	apiVersion string
	pl         runtime.Pipeline
}

// ClientOptions configures the toolboxes client.
type ClientOptions struct {
	azcore.ClientOptions
	APIVersion string
}

// NewClient constructs a toolboxes client targeting endpoint.
func NewClient(endpoint string, cred azcore.TokenCredential, opts *ClientOptions) (*Client, error) {
	if endpoint == "" {
		return nil, errors.New("toolboxes: endpoint is required")
	}
	if cred == nil {
		return nil, errors.New("toolboxes: cred is required")
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
	Limit  *int32
	Order  *PageOrder
	After  *string
	Before *string
}

// NewListPager returns a Pager that issues GET /toolboxes.
func (c *Client) NewListPager(opts *ListOptions) *runtime.Pager[ToolboxesPage] {
	first := true
	return runtime.NewPager(runtime.PagingHandler[ToolboxesPage]{
		More: func(page ToolboxesPage) bool {
			return page.HasMore && page.LastID != ""
		},
		Fetcher: func(ctx context.Context, page *ToolboxesPage) (ToolboxesPage, error) {
			req, err := c.newGet(ctx, "/toolboxes")
			if err != nil {
				return ToolboxesPage{}, err
			}
			c.applyListQuery(req, opts, page, &first)
			body, err := c.do(req, http.StatusOK)
			if err != nil {
				return ToolboxesPage{}, err
			}
			var out ToolboxesPage
			if err := json.Unmarshal(body, &out); err != nil {
				return ToolboxesPage{}, fmt.Errorf("toolboxes.List: decode: %w", err)
			}
			return out, nil
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

// NewListVersionsPager returns a Pager that issues GET /toolboxes/{name}/versions.
func (c *Client) NewListVersionsPager(toolboxName string, opts *ListVersionsOptions) *runtime.Pager[ToolboxVersionsPage] {
	first := true
	return runtime.NewPager(runtime.PagingHandler[ToolboxVersionsPage]{
		More: func(page ToolboxVersionsPage) bool {
			return page.HasMore && page.LastID != ""
		},
		Fetcher: func(ctx context.Context, page *ToolboxVersionsPage) (ToolboxVersionsPage, error) {
			if toolboxName == "" {
				return ToolboxVersionsPage{}, errors.New("toolboxes.NewListVersionsPager: toolboxName is required")
			}
			req, err := c.newGet(ctx, fmt.Sprintf("/toolboxes/%s/versions", toolboxName))
			if err != nil {
				return ToolboxVersionsPage{}, err
			}
			lo := &ListOptions{}
			if opts != nil {
				lo.Limit, lo.Order, lo.After, lo.Before = opts.Limit, opts.Order, opts.After, opts.Before
			}
			// Reuse cursor query application via a local page wrapper.
			c.applyVersionsListQuery(req, lo, page, &first)
			body, err := c.do(req, http.StatusOK)
			if err != nil {
				return ToolboxVersionsPage{}, err
			}
			var out ToolboxVersionsPage
			if err := json.Unmarshal(body, &out); err != nil {
				return ToolboxVersionsPage{}, fmt.Errorf("toolboxes.ListVersions: decode: %w", err)
			}
			return out, nil
		},
	})
}

// GetOptions is the optional parameter set for Get.
type GetOptions struct{}

// Get retrieves a toolbox by name.
func (c *Client) Get(ctx context.Context, toolboxName string, _ *GetOptions) (ToolboxObject, error) {
	if toolboxName == "" {
		return ToolboxObject{}, errors.New("toolboxes.Get: toolboxName is required")
	}
	body, err := c.jsonRequest(ctx, http.MethodGet,
		fmt.Sprintf("%s/toolboxes/%s", c.endpoint, toolboxName), nil, "", http.StatusOK)
	if err != nil {
		return ToolboxObject{}, err
	}
	var out ToolboxObject
	if err := json.Unmarshal(body, &out); err != nil {
		return ToolboxObject{}, fmt.Errorf("toolboxes.Get: decode: %w", err)
	}
	return out, nil
}

// GetVersionOptions is the optional parameter set for GetVersion.
type GetVersionOptions struct{}

// GetVersion retrieves a specific version of a toolbox.
func (c *Client) GetVersion(ctx context.Context, toolboxName, version string, _ *GetVersionOptions) (ToolboxVersionObject, error) {
	if toolboxName == "" || version == "" {
		return ToolboxVersionObject{}, errors.New("toolboxes.GetVersion: toolboxName and version are required")
	}
	body, err := c.jsonRequest(ctx, http.MethodGet,
		fmt.Sprintf("%s/toolboxes/%s/versions/%s", c.endpoint, toolboxName, version),
		nil, "", http.StatusOK)
	if err != nil {
		return ToolboxVersionObject{}, err
	}
	var out ToolboxVersionObject
	if err := json.Unmarshal(body, &out); err != nil {
		return ToolboxVersionObject{}, fmt.Errorf("toolboxes.GetVersion: decode: %w", err)
	}
	return out, nil
}

// UpdateOptions is the optional parameter set for Update.
type UpdateOptions struct{}

// Update points a toolbox at a different default version.
// PATCH /toolboxes/{name} with {"default_version": defaultVersion}.
func (c *Client) Update(ctx context.Context, toolboxName, defaultVersion string, _ *UpdateOptions) (ToolboxObject, error) {
	if toolboxName == "" || defaultVersion == "" {
		return ToolboxObject{}, errors.New("toolboxes.Update: toolboxName and defaultVersion are required")
	}
	payload, err := json.Marshal(UpdateBody{DefaultVersion: defaultVersion})
	if err != nil {
		return ToolboxObject{}, fmt.Errorf("toolboxes.Update: marshal: %w", err)
	}
	body, err := c.jsonRequest(ctx, http.MethodPatch,
		fmt.Sprintf("%s/toolboxes/%s", c.endpoint, toolboxName),
		payload, "application/json", http.StatusOK)
	if err != nil {
		return ToolboxObject{}, err
	}
	var out ToolboxObject
	if err := json.Unmarshal(body, &out); err != nil {
		return ToolboxObject{}, fmt.Errorf("toolboxes.Update: decode: %w", err)
	}
	return out, nil
}

// CreateVersionOptions is the optional parameter set for CreateVersion.
type CreateVersionOptions struct{}

// CreateVersion creates a new version of a toolbox. If the toolbox does not exist, it is created.
// POST /toolboxes/{name}/versions returns 200.
func (c *Client) CreateVersion(ctx context.Context, toolboxName string, body CreateVersionBody, _ *CreateVersionOptions) (ToolboxVersionObject, error) {
	if toolboxName == "" {
		return ToolboxVersionObject{}, errors.New("toolboxes.CreateVersion: toolboxName is required")
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return ToolboxVersionObject{}, fmt.Errorf("toolboxes.CreateVersion: marshal: %w", err)
	}
	respBody, err := c.jsonRequest(ctx, http.MethodPost,
		fmt.Sprintf("%s/toolboxes/%s/versions", c.endpoint, toolboxName),
		payload, "application/json", http.StatusOK)
	if err != nil {
		return ToolboxVersionObject{}, err
	}
	var out ToolboxVersionObject
	if err := json.Unmarshal(respBody, &out); err != nil {
		return ToolboxVersionObject{}, fmt.Errorf("toolboxes.CreateVersion: decode: %w", err)
	}
	return out, nil
}

// DeleteOptions is the optional parameter set for Delete.
type DeleteOptions struct{}

// DeleteResponse is empty; Delete is success/failure.
type DeleteResponse struct{}

// Delete removes a toolbox and all its versions.
// DELETE /toolboxes/{name} returns 204.
func (c *Client) Delete(ctx context.Context, toolboxName string, _ *DeleteOptions) (DeleteResponse, error) {
	if toolboxName == "" {
		return DeleteResponse{}, errors.New("toolboxes.Delete: toolboxName is required")
	}
	if _, err := c.jsonRequest(ctx, http.MethodDelete,
		fmt.Sprintf("%s/toolboxes/%s", c.endpoint, toolboxName),
		nil, "", http.StatusNoContent); err != nil {
		return DeleteResponse{}, err
	}
	return DeleteResponse{}, nil
}

// DeleteVersionOptions is the optional parameter set for DeleteVersion.
type DeleteVersionOptions struct{}

// DeleteVersion removes a specific version of a toolbox.
// DELETE /toolboxes/{name}/versions/{version} returns 204.
func (c *Client) DeleteVersion(ctx context.Context, toolboxName, version string, _ *DeleteVersionOptions) (DeleteResponse, error) {
	if toolboxName == "" || version == "" {
		return DeleteResponse{}, errors.New("toolboxes.DeleteVersion: toolboxName and version are required")
	}
	if _, err := c.jsonRequest(ctx, http.MethodDelete,
		fmt.Sprintf("%s/toolboxes/%s/versions/%s", c.endpoint, toolboxName, version),
		nil, "", http.StatusNoContent); err != nil {
		return DeleteResponse{}, err
	}
	return DeleteResponse{}, nil
}

// --- helpers ---

func (c *Client) newGet(ctx context.Context, path string) (*policy.Request, error) {
	req, err := runtime.NewRequest(ctx, http.MethodGet, c.endpoint+path)
	if err != nil {
		return nil, err
	}
	req.Raw().Header.Set("foundry-features", foundryHeader)
	req.Raw().Header.Set("Accept", "application/json")
	return req, nil
}

func (c *Client) applyListQuery(req *policy.Request, opts *ListOptions, page *ToolboxesPage, first *bool) {
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
	case *first:
		*first = false
		if opts != nil && opts.After != nil {
			q.Set("after", *opts.After)
		}
	case page != nil:
		q.Set("after", page.LastID)
	}
	q.Set("api-version", c.apiVersion)
	req.Raw().URL.RawQuery = q.Encode()
}

func (c *Client) applyVersionsListQuery(req *policy.Request, opts *ListOptions, page *ToolboxVersionsPage, first *bool) {
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
	case *first:
		*first = false
		if opts != nil && opts.After != nil {
			q.Set("after", *opts.After)
		}
	case page != nil:
		q.Set("after", page.LastID)
	}
	q.Set("api-version", c.apiVersion)
	req.Raw().URL.RawQuery = q.Encode()
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
