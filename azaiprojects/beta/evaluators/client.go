package evaluators

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
	moduleName    = "azaiprojects/beta/evaluators"
	moduleVersion = "0.1.0"
	defaultScope  = "https://ai.azure.com/.default"
	defaultAPIVer = "v1"
	foundryHeader = "Evaluations=V1Preview"
)

// Client provides the beta.evaluators operation group.
type Client struct {
	endpoint   string
	apiVersion string
	pl         runtime.Pipeline
}

// ClientOptions configures the evaluators client.
type ClientOptions struct {
	azcore.ClientOptions
	APIVersion string
}

// NewClient constructs an evaluators client targeting endpoint.
func NewClient(endpoint string, cred azcore.TokenCredential, opts *ClientOptions) (*Client, error) {
	if endpoint == "" {
		return nil, errors.New("evaluators: endpoint is required")
	}
	if cred == nil {
		return nil, errors.New("evaluators: cred is required")
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

// ListOptions is the optional parameter set for NewListPager.
type ListOptions struct {
	EvaluatorType *string
	Limit         *int32
}

// NewListPager returns a Pager that issues GET /evaluators.
func (c *Client) NewListPager(opts *ListOptions) *runtime.Pager[EvaluatorVersionsPage] {
	first := true
	return runtime.NewPager(runtime.PagingHandler[EvaluatorVersionsPage]{
		More: func(page EvaluatorVersionsPage) bool {
			return page.NextLink != nil && *page.NextLink != ""
		},
		Fetcher: func(ctx context.Context, page *EvaluatorVersionsPage) (EvaluatorVersionsPage, error) {
			var url string
			if first || page == nil || page.NextLink == nil {
				first = false
				url = c.endpoint + "/evaluators"
			} else {
				url = *page.NextLink
			}
			req, err := runtime.NewRequest(ctx, http.MethodGet, url)
			if err != nil {
				return EvaluatorVersionsPage{}, err
			}
			req.Raw().Header.Set("foundry-features", foundryHeader)
			req.Raw().Header.Set("Accept", "application/json")
			q := req.Raw().URL.Query()
			if opts != nil {
				if opts.EvaluatorType != nil {
					q.Set("type", *opts.EvaluatorType)
				}
				if opts.Limit != nil {
					q.Set("limit", strconv.FormatInt(int64(*opts.Limit), 10))
				}
			}
			q.Set("api-version", c.apiVersion)
			req.Raw().URL.RawQuery = q.Encode()
			body, err := c.do(req, http.StatusOK)
			if err != nil {
				return EvaluatorVersionsPage{}, err
			}
			var out EvaluatorVersionsPage
			if err := json.Unmarshal(body, &out); err != nil {
				return EvaluatorVersionsPage{}, fmt.Errorf("evaluators.List: decode: %w", err)
			}
			return out, nil
		},
	})
}

// ListVersionsOptions is the optional parameter set for NewListVersionsPager.
type ListVersionsOptions struct {
	EvaluatorType *string
	Limit         *int32
}

// NewListVersionsPager returns a Pager for GET /evaluators/{name}/versions.
func (c *Client) NewListVersionsPager(name string, opts *ListVersionsOptions) *runtime.Pager[EvaluatorVersionsPage] {
	first := true
	return runtime.NewPager(runtime.PagingHandler[EvaluatorVersionsPage]{
		More: func(page EvaluatorVersionsPage) bool {
			return page.NextLink != nil && *page.NextLink != ""
		},
		Fetcher: func(ctx context.Context, page *EvaluatorVersionsPage) (EvaluatorVersionsPage, error) {
			if name == "" {
				return EvaluatorVersionsPage{}, errors.New("evaluators.ListVersions: name is required")
			}
			var url string
			if first || page == nil || page.NextLink == nil {
				first = false
				url = fmt.Sprintf("%s/evaluators/%s/versions", c.endpoint, name)
			} else {
				url = *page.NextLink
			}
			req, err := runtime.NewRequest(ctx, http.MethodGet, url)
			if err != nil {
				return EvaluatorVersionsPage{}, err
			}
			req.Raw().Header.Set("foundry-features", foundryHeader)
			req.Raw().Header.Set("Accept", "application/json")
			q := req.Raw().URL.Query()
			if opts != nil {
				if opts.EvaluatorType != nil {
					q.Set("type", *opts.EvaluatorType)
				}
				if opts.Limit != nil {
					q.Set("limit", strconv.FormatInt(int64(*opts.Limit), 10))
				}
			}
			q.Set("api-version", c.apiVersion)
			req.Raw().URL.RawQuery = q.Encode()
			body, err := c.do(req, http.StatusOK)
			if err != nil {
				return EvaluatorVersionsPage{}, err
			}
			var out EvaluatorVersionsPage
			if err := json.Unmarshal(body, &out); err != nil {
				return EvaluatorVersionsPage{}, fmt.Errorf("evaluators.ListVersions: decode: %w", err)
			}
			return out, nil
		},
	})
}

// GetVersionOptions is the optional parameter set for GetVersion.
type GetVersionOptions struct{}

// GetVersion retrieves a specific evaluator version.
func (c *Client) GetVersion(ctx context.Context, name, version string, _ *GetVersionOptions) (EvaluatorVersion, error) {
	if name == "" {
		return EvaluatorVersion{}, errors.New("evaluators.GetVersion: name is required")
	}
	if version == "" {
		return EvaluatorVersion{}, errors.New("evaluators.GetVersion: version is required")
	}
	body, err := c.jsonRequest(ctx, http.MethodGet,
		fmt.Sprintf("%s/evaluators/%s/versions/%s", c.endpoint, name, version), nil, "", http.StatusOK)
	if err != nil {
		return EvaluatorVersion{}, err
	}
	var out EvaluatorVersion
	if err := json.Unmarshal(body, &out); err != nil {
		return EvaluatorVersion{}, fmt.Errorf("evaluators.GetVersion: decode: %w", err)
	}
	return out, nil
}

// CreateVersionOptions is the optional parameter set for CreateVersion.
type CreateVersionOptions struct{}

// CreateVersion creates a new evaluator version.
// POST /evaluators/{name}/versions returns 201.
func (c *Client) CreateVersion(ctx context.Context, name string, ev EvaluatorVersion, _ *CreateVersionOptions) (EvaluatorVersion, error) {
	if name == "" {
		return EvaluatorVersion{}, errors.New("evaluators.CreateVersion: name is required")
	}
	payload, err := json.Marshal(ev)
	if err != nil {
		return EvaluatorVersion{}, fmt.Errorf("evaluators.CreateVersion: marshal: %w", err)
	}
	body, err := c.jsonRequest(ctx, http.MethodPost,
		fmt.Sprintf("%s/evaluators/%s/versions", c.endpoint, name), payload, "application/json", http.StatusCreated)
	if err != nil {
		return EvaluatorVersion{}, err
	}
	var out EvaluatorVersion
	if err := json.Unmarshal(body, &out); err != nil {
		return EvaluatorVersion{}, fmt.Errorf("evaluators.CreateVersion: decode: %w", err)
	}
	return out, nil
}

// UpdateVersionOptions is the optional parameter set for UpdateVersion.
type UpdateVersionOptions struct{}

// UpdateVersion patches an existing evaluator version.
// PATCH /evaluators/{name}/versions/{version} returns 200.
func (c *Client) UpdateVersion(ctx context.Context, name, version string, ev EvaluatorVersion, _ *UpdateVersionOptions) (EvaluatorVersion, error) {
	if name == "" {
		return EvaluatorVersion{}, errors.New("evaluators.UpdateVersion: name is required")
	}
	if version == "" {
		return EvaluatorVersion{}, errors.New("evaluators.UpdateVersion: version is required")
	}
	payload, err := json.Marshal(ev)
	if err != nil {
		return EvaluatorVersion{}, fmt.Errorf("evaluators.UpdateVersion: marshal: %w", err)
	}
	body, err := c.jsonRequest(ctx, http.MethodPatch,
		fmt.Sprintf("%s/evaluators/%s/versions/%s", c.endpoint, name, version), payload, "application/json", http.StatusOK)
	if err != nil {
		return EvaluatorVersion{}, err
	}
	var out EvaluatorVersion
	if err := json.Unmarshal(body, &out); err != nil {
		return EvaluatorVersion{}, fmt.Errorf("evaluators.UpdateVersion: decode: %w", err)
	}
	return out, nil
}

// DeleteVersionOptions is the optional parameter set for DeleteVersion.
type DeleteVersionOptions struct{}

// DeleteVersionResponse is the (empty) response of DeleteVersion.
type DeleteVersionResponse struct{}

// DeleteVersion deletes a specific evaluator version.
// DELETE /evaluators/{name}/versions/{version} returns 204.
func (c *Client) DeleteVersion(ctx context.Context, name, version string, _ *DeleteVersionOptions) (DeleteVersionResponse, error) {
	if name == "" {
		return DeleteVersionResponse{}, errors.New("evaluators.DeleteVersion: name is required")
	}
	if version == "" {
		return DeleteVersionResponse{}, errors.New("evaluators.DeleteVersion: version is required")
	}
	if _, err := c.jsonRequest(ctx, http.MethodDelete,
		fmt.Sprintf("%s/evaluators/%s/versions/%s", c.endpoint, name, version), nil, "", http.StatusNoContent); err != nil {
		return DeleteVersionResponse{}, err
	}
	return DeleteVersionResponse{}, nil
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
