package evaluationtaxonomies

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
)

const (
	moduleName    = "azaiprojects/beta/evaluationtaxonomies"
	moduleVersion = "0.1.0"
	defaultScope  = "https://ai.azure.com/.default"
	defaultAPIVer = "v1"
	foundryHeader = "Evaluations=V1Preview"
)

// Client provides the beta.evaluationTaxonomies operation group.
type Client struct {
	endpoint   string
	apiVersion string
	pl         runtime.Pipeline
}

// ClientOptions configures the evaluationtaxonomies client.
type ClientOptions struct {
	azcore.ClientOptions
	APIVersion string
}

// NewClient constructs a client targeting endpoint.
func NewClient(endpoint string, cred azcore.TokenCredential, opts *ClientOptions) (*Client, error) {
	if endpoint == "" {
		return nil, errors.New("evaluationtaxonomies: endpoint is required")
	}
	if cred == nil {
		return nil, errors.New("evaluationtaxonomies: cred is required")
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
	InputName       *string
	InputType       *string
	ClientRequestID string
}

// NewListPager returns a Pager that issues GET /evaluationtaxonomies.
func (c *Client) NewListPager(opts *ListOptions) *runtime.Pager[EvaluationTaxonomiesPage] {
	first := true
	return runtime.NewPager(runtime.PagingHandler[EvaluationTaxonomiesPage]{
		More: func(page EvaluationTaxonomiesPage) bool {
			return page.NextLink != nil && *page.NextLink != ""
		},
		Fetcher: func(ctx context.Context, page *EvaluationTaxonomiesPage) (EvaluationTaxonomiesPage, error) {
			var url string
			if first || page == nil || page.NextLink == nil {
				first = false
				url = c.endpoint + "/evaluationtaxonomies"
			} else {
				url = *page.NextLink
			}
			req, err := runtime.NewRequest(ctx, http.MethodGet, url)
			if err != nil {
				return EvaluationTaxonomiesPage{}, err
			}
			req.Raw().Header.Set("foundry-features", foundryHeader)
			req.Raw().Header.Set("Accept", "application/json")
			if opts != nil && opts.ClientRequestID != "" {
				req.Raw().Header.Set("x-ms-client-request-id", opts.ClientRequestID)
			}
			q := req.Raw().URL.Query()
			if opts != nil {
				if opts.InputName != nil {
					q.Set("inputName", *opts.InputName)
				}
				if opts.InputType != nil {
					q.Set("inputType", *opts.InputType)
				}
			}
			q.Set("api-version", c.apiVersion)
			req.Raw().URL.RawQuery = q.Encode()
			body, err := c.do(req, http.StatusOK)
			if err != nil {
				return EvaluationTaxonomiesPage{}, err
			}
			var out EvaluationTaxonomiesPage
			if err := json.Unmarshal(body, &out); err != nil {
				return EvaluationTaxonomiesPage{}, fmt.Errorf("evaluationtaxonomies.List: decode: %w", err)
			}
			return out, nil
		},
	})
}

// GetOptions is the optional parameter set for Get.
type GetOptions struct {
	ClientRequestID string
}

// Get retrieves a taxonomy by name.
func (c *Client) Get(ctx context.Context, name string, opts *GetOptions) (EvaluationTaxonomy, error) {
	if name == "" {
		return EvaluationTaxonomy{}, errors.New("evaluationtaxonomies.Get: name is required")
	}
	req, err := runtime.NewRequest(ctx, http.MethodGet,
		fmt.Sprintf("%s/evaluationtaxonomies/%s", c.endpoint, name))
	if err != nil {
		return EvaluationTaxonomy{}, err
	}
	req.Raw().Header.Set("foundry-features", foundryHeader)
	req.Raw().Header.Set("Accept", "application/json")
	if opts != nil && opts.ClientRequestID != "" {
		req.Raw().Header.Set("x-ms-client-request-id", opts.ClientRequestID)
	}
	q := req.Raw().URL.Query()
	q.Set("api-version", c.apiVersion)
	req.Raw().URL.RawQuery = q.Encode()
	body, err := c.do(req, http.StatusOK)
	if err != nil {
		return EvaluationTaxonomy{}, err
	}
	var out EvaluationTaxonomy
	if err := json.Unmarshal(body, &out); err != nil {
		return EvaluationTaxonomy{}, fmt.Errorf("evaluationtaxonomies.Get: decode: %w", err)
	}
	return out, nil
}

// CreateOptions is the optional parameter set for Create.
type CreateOptions struct{}

// Create creates a taxonomy.
// PUT /evaluationtaxonomies/{name} accepts 200 or 201.
func (c *Client) Create(ctx context.Context, name string, body EvaluationTaxonomy, _ *CreateOptions) (EvaluationTaxonomy, error) {
	if name == "" {
		return EvaluationTaxonomy{}, errors.New("evaluationtaxonomies.Create: name is required")
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return EvaluationTaxonomy{}, fmt.Errorf("evaluationtaxonomies.Create: marshal: %w", err)
	}
	respBody, err := c.jsonRequest(ctx, http.MethodPut,
		fmt.Sprintf("%s/evaluationtaxonomies/%s", c.endpoint, name), payload, "application/json",
		http.StatusOK, http.StatusCreated)
	if err != nil {
		return EvaluationTaxonomy{}, err
	}
	var out EvaluationTaxonomy
	if err := json.Unmarshal(respBody, &out); err != nil {
		return EvaluationTaxonomy{}, fmt.Errorf("evaluationtaxonomies.Create: decode: %w", err)
	}
	return out, nil
}

// UpdateOptions is the optional parameter set for Update.
type UpdateOptions struct{}

// Update patches an existing taxonomy.
// PATCH /evaluationtaxonomies/{name} returns 200.
func (c *Client) Update(ctx context.Context, name string, body EvaluationTaxonomy, _ *UpdateOptions) (EvaluationTaxonomy, error) {
	if name == "" {
		return EvaluationTaxonomy{}, errors.New("evaluationtaxonomies.Update: name is required")
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return EvaluationTaxonomy{}, fmt.Errorf("evaluationtaxonomies.Update: marshal: %w", err)
	}
	respBody, err := c.jsonRequest(ctx, http.MethodPatch,
		fmt.Sprintf("%s/evaluationtaxonomies/%s", c.endpoint, name), payload, "application/json", http.StatusOK)
	if err != nil {
		return EvaluationTaxonomy{}, err
	}
	var out EvaluationTaxonomy
	if err := json.Unmarshal(respBody, &out); err != nil {
		return EvaluationTaxonomy{}, fmt.Errorf("evaluationtaxonomies.Update: decode: %w", err)
	}
	return out, nil
}

// DeleteOptions is the optional parameter set for Delete.
type DeleteOptions struct {
	ClientRequestID string
}

// DeleteResponse is the (empty) response of Delete.
type DeleteResponse struct{}

// Delete deletes a taxonomy by name.
// DELETE /evaluationtaxonomies/{name} returns 204.
func (c *Client) Delete(ctx context.Context, name string, opts *DeleteOptions) (DeleteResponse, error) {
	if name == "" {
		return DeleteResponse{}, errors.New("evaluationtaxonomies.Delete: name is required")
	}
	req, err := runtime.NewRequest(ctx, http.MethodDelete,
		fmt.Sprintf("%s/evaluationtaxonomies/%s", c.endpoint, name))
	if err != nil {
		return DeleteResponse{}, err
	}
	req.Raw().Header.Set("foundry-features", foundryHeader)
	if opts != nil && opts.ClientRequestID != "" {
		req.Raw().Header.Set("x-ms-client-request-id", opts.ClientRequestID)
	}
	q := req.Raw().URL.Query()
	q.Set("api-version", c.apiVersion)
	req.Raw().URL.RawQuery = q.Encode()
	if _, err := c.do(req, http.StatusNoContent); err != nil {
		return DeleteResponse{}, err
	}
	return DeleteResponse{}, nil
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
