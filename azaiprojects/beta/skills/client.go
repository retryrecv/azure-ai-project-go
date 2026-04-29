package skills

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
	moduleName    = "azaiprojects/beta/skills"
	moduleVersion = "0.1.0"
	defaultScope  = "https://ai.azure.com/.default"
	defaultAPIVer = "v1"
	foundryHeader = "Skills=V1Preview"
)

// Client provides the beta.skills operation group.
type Client struct {
	endpoint   string
	apiVersion string
	pl         runtime.Pipeline
}

// ClientOptions configures the skills client.
type ClientOptions struct {
	azcore.ClientOptions
	APIVersion string
}

// NewClient constructs a skills client targeting endpoint.
func NewClient(endpoint string, cred azcore.TokenCredential, opts *ClientOptions) (*Client, error) {
	if endpoint == "" {
		return nil, errors.New("skills: endpoint is required")
	}
	if cred == nil {
		return nil, errors.New("skills: cred is required")
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

// NewListPager returns a Pager that issues GET /skills.
func (c *Client) NewListPager(opts *ListOptions) *runtime.Pager[SkillsPage] {
	first := true
	return runtime.NewPager(runtime.PagingHandler[SkillsPage]{
		More: func(page SkillsPage) bool {
			return page.HasMore && page.LastID != ""
		},
		Fetcher: func(ctx context.Context, page *SkillsPage) (SkillsPage, error) {
			req, err := runtime.NewRequest(ctx, http.MethodGet, c.endpoint+"/skills")
			if err != nil {
				return SkillsPage{}, err
			}
			req.Raw().Header.Set("foundry-features", foundryHeader)
			req.Raw().Header.Set("Accept", "application/json")
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
			body, err := c.do(req, http.StatusOK)
			if err != nil {
				return SkillsPage{}, err
			}
			var out SkillsPage
			if err := json.Unmarshal(body, &out); err != nil {
				return SkillsPage{}, fmt.Errorf("skills.List: decode: %w", err)
			}
			return out, nil
		},
	})
}

// GetOptions is the optional parameter set for Get.
type GetOptions struct{}

// Get retrieves a skill by name.
func (c *Client) Get(ctx context.Context, name string, _ *GetOptions) (SkillObject, error) {
	if name == "" {
		return SkillObject{}, errors.New("skills.Get: name is required")
	}
	body, err := c.jsonRequest(ctx, http.MethodGet,
		fmt.Sprintf("%s/skills/%s", c.endpoint, name), nil, "", http.StatusOK)
	if err != nil {
		return SkillObject{}, err
	}
	var out SkillObject
	if err := json.Unmarshal(body, &out); err != nil {
		return SkillObject{}, fmt.Errorf("skills.Get: decode: %w", err)
	}
	return out, nil
}

// CreateOptions is the optional parameter set for Create.
type CreateOptions struct{}

// Create creates a new skill.
// POST /skills returns 201.
func (c *Client) Create(ctx context.Context, body CreateSkillBody, _ *CreateOptions) (SkillObject, error) {
	if body.Name == "" {
		return SkillObject{}, errors.New("skills.Create: body.Name is required")
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return SkillObject{}, fmt.Errorf("skills.Create: marshal: %w", err)
	}
	respBody, err := c.jsonRequest(ctx, http.MethodPost,
		c.endpoint+"/skills", payload, "application/json", http.StatusCreated)
	if err != nil {
		return SkillObject{}, err
	}
	var out SkillObject
	if err := json.Unmarshal(respBody, &out); err != nil {
		return SkillObject{}, fmt.Errorf("skills.Create: decode: %w", err)
	}
	return out, nil
}

// UpdateOptions is the optional parameter set for Update.
type UpdateOptions struct{}

// Update updates an existing skill.
// POST /skills/{name} returns 200.
func (c *Client) Update(ctx context.Context, name string, body UpdateSkillBody, _ *UpdateOptions) (SkillObject, error) {
	if name == "" {
		return SkillObject{}, errors.New("skills.Update: name is required")
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return SkillObject{}, fmt.Errorf("skills.Update: marshal: %w", err)
	}
	respBody, err := c.jsonRequest(ctx, http.MethodPost,
		fmt.Sprintf("%s/skills/%s", c.endpoint, name), payload, "application/json", http.StatusOK)
	if err != nil {
		return SkillObject{}, err
	}
	var out SkillObject
	if err := json.Unmarshal(respBody, &out); err != nil {
		return SkillObject{}, fmt.Errorf("skills.Update: decode: %w", err)
	}
	return out, nil
}

// DeleteOptions is the optional parameter set for Delete.
type DeleteOptions struct{}

// Delete deletes a skill by name.
// DELETE /skills/{name} returns 200 with body {name, deleted}.
func (c *Client) Delete(ctx context.Context, name string, _ *DeleteOptions) (DeleteSkillResponse, error) {
	if name == "" {
		return DeleteSkillResponse{}, errors.New("skills.Delete: name is required")
	}
	respBody, err := c.jsonRequest(ctx, http.MethodDelete,
		fmt.Sprintf("%s/skills/%s", c.endpoint, name), nil, "", http.StatusOK)
	if err != nil {
		return DeleteSkillResponse{}, err
	}
	var out DeleteSkillResponse
	if err := json.Unmarshal(respBody, &out); err != nil {
		return DeleteSkillResponse{}, fmt.Errorf("skills.Delete: decode: %w", err)
	}
	return out, nil
}

// DownloadOptions is the optional parameter set for Download.
type DownloadOptions struct{}

// DownloadResponse wraps the binary skill package body. Caller must Close the body.
type DownloadResponse struct {
	Body io.ReadCloser
}

// Download fetches a skill's package as a binary stream (zip).
// GET /skills/{name}:download returns 200 with application/zip body.
func (c *Client) Download(ctx context.Context, name string, _ *DownloadOptions) (DownloadResponse, error) {
	if name == "" {
		return DownloadResponse{}, errors.New("skills.Download: name is required")
	}
	req, err := runtime.NewRequest(ctx, http.MethodGet,
		fmt.Sprintf("%s/skills/%s:download", c.endpoint, name))
	if err != nil {
		return DownloadResponse{}, err
	}
	req.Raw().Header.Set("foundry-features", foundryHeader)
	req.Raw().Header.Set("Accept", "application/zip")
	q := req.Raw().URL.Query()
	q.Set("api-version", c.apiVersion)
	req.Raw().URL.RawQuery = q.Encode()
	resp, err := c.pl.Do(req)
	if err != nil {
		return DownloadResponse{}, err
	}
	if !runtime.HasStatusCode(resp, http.StatusOK) {
		return DownloadResponse{}, runtime.NewResponseError(resp)
	}
	return DownloadResponse{Body: resp.Body}, nil
}

// CreateFromPackageOptions is the optional parameter set for CreateFromPackage.
type CreateFromPackageOptions struct{}

// CreateFromPackage creates a skill from a zip package.
// POST /skills:import returns 201.
func (c *Client) CreateFromPackage(ctx context.Context, pkg []byte, _ *CreateFromPackageOptions) (SkillObject, error) {
	if len(pkg) == 0 {
		return SkillObject{}, errors.New("skills.CreateFromPackage: pkg is required")
	}
	respBody, err := c.jsonRequest(ctx, http.MethodPost,
		c.endpoint+"/skills:import", pkg, "application/zip", http.StatusCreated)
	if err != nil {
		return SkillObject{}, err
	}
	var out SkillObject
	if err := json.Unmarshal(respBody, &out); err != nil {
		return SkillObject{}, fmt.Errorf("skills.CreateFromPackage: decode: %w", err)
	}
	return out, nil
}

// jsonRequest issues a request with the foundry-features header,
// optional body, api-version query param, and validates the status.
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

// do is for callers that handle the response body themselves (used by NewListPager).
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
