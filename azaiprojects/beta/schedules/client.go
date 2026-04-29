package schedules

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

	"github.com/sambo/ai-projects-go/azaiprojects/internal/shared"
)

const (
	moduleName    = "azaiprojects/beta/schedules"
	moduleVersion = "0.1.0"
	defaultScope  = "https://ai.azure.com/.default"
	defaultAPIVer = "v1"
	foundryHeader = "Schedules=V1Preview"
)

// Client provides the beta.schedules operation group.
type Client struct {
	endpoint   string
	apiVersion string
	pl         runtime.Pipeline
}

// ClientOptions configures the schedules client.
type ClientOptions struct {
	azcore.ClientOptions
	APIVersion string
}

// NewClient constructs a schedules client targeting endpoint.
func NewClient(endpoint string, cred azcore.TokenCredential, opts *ClientOptions) (*Client, error) {
	if endpoint == "" {
		return nil, errors.New("schedules: endpoint is required")
	}
	if cred == nil {
		return nil, errors.New("schedules: cred is required")
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

// ListResponse is one page of Schedule results.
type ListResponse struct {
	shared.PageResponse[Schedule]
}

// ListOptions is the optional parameter set for NewListPager.
type ListOptions struct {
	Type    *string
	Enabled *bool
}

// NewListPager returns a Pager that issues GET /schedules.
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
				url = c.endpoint + "/schedules"
			} else {
				url = *page.NextLink
			}
			req, err := runtime.NewRequest(ctx, http.MethodGet, url)
			if err != nil {
				return ListResponse{}, err
			}
			req.Raw().Header.Set("foundry-features", foundryHeader)
			req.Raw().Header.Set("Accept", "application/json")
			q := req.Raw().URL.Query()
			if opts != nil {
				if opts.Type != nil {
					q.Set("type", *opts.Type)
				}
				if opts.Enabled != nil {
					q.Set("enabled", strconv.FormatBool(*opts.Enabled))
				}
			}
			q.Set("api-version", c.apiVersion)
			req.Raw().URL.RawQuery = q.Encode()
			body, err := c.do(req, http.StatusOK)
			if err != nil {
				return ListResponse{}, err
			}
			var out ListResponse
			if err := json.Unmarshal(body, &out); err != nil {
				return ListResponse{}, fmt.Errorf("schedules.List: decode: %w", err)
			}
			return out, nil
		},
	})
}

// ListRunsResponse is one page of ScheduleRun results.
type ListRunsResponse struct {
	shared.PageResponse[ScheduleRun]
}

// ListRunsOptions is the optional parameter set for NewListRunsPager.
type ListRunsOptions struct {
	Type    *string
	Enabled *bool
}

// NewListRunsPager returns a Pager that issues GET /schedules/{id}/runs.
func (c *Client) NewListRunsPager(id string, opts *ListRunsOptions) *runtime.Pager[ListRunsResponse] {
	first := true
	return runtime.NewPager(runtime.PagingHandler[ListRunsResponse]{
		More: func(page ListRunsResponse) bool {
			return page.NextLink != nil && *page.NextLink != ""
		},
		Fetcher: func(ctx context.Context, page *ListRunsResponse) (ListRunsResponse, error) {
			if id == "" {
				return ListRunsResponse{}, errors.New("schedules.NewListRunsPager: id is required")
			}
			var url string
			if first || page == nil || page.NextLink == nil {
				first = false
				url = fmt.Sprintf("%s/schedules/%s/runs", c.endpoint, id)
			} else {
				url = *page.NextLink
			}
			req, err := runtime.NewRequest(ctx, http.MethodGet, url)
			if err != nil {
				return ListRunsResponse{}, err
			}
			req.Raw().Header.Set("foundry-features", foundryHeader)
			req.Raw().Header.Set("Accept", "application/json")
			q := req.Raw().URL.Query()
			if opts != nil {
				if opts.Type != nil {
					q.Set("type", *opts.Type)
				}
				if opts.Enabled != nil {
					q.Set("enabled", strconv.FormatBool(*opts.Enabled))
				}
			}
			q.Set("api-version", c.apiVersion)
			req.Raw().URL.RawQuery = q.Encode()
			body, err := c.do(req, http.StatusOK)
			if err != nil {
				return ListRunsResponse{}, err
			}
			var out ListRunsResponse
			if err := json.Unmarshal(body, &out); err != nil {
				return ListRunsResponse{}, fmt.Errorf("schedules.ListRuns: decode: %w", err)
			}
			return out, nil
		},
	})
}

// GetOptions is the optional parameter set for Get.
type GetOptions struct{}

// Get retrieves a schedule by id.
func (c *Client) Get(ctx context.Context, id string, _ *GetOptions) (Schedule, error) {
	if id == "" {
		return Schedule{}, errors.New("schedules.Get: id is required")
	}
	body, err := c.jsonRequest(ctx, http.MethodGet,
		fmt.Sprintf("%s/schedules/%s", c.endpoint, id), nil, "", http.StatusOK)
	if err != nil {
		return Schedule{}, err
	}
	var out Schedule
	if err := json.Unmarshal(body, &out); err != nil {
		return Schedule{}, fmt.Errorf("schedules.Get: decode: %w", err)
	}
	return out, nil
}

// GetRunOptions is the optional parameter set for GetRun.
type GetRunOptions struct{}

// GetRun retrieves a schedule run by run id.
func (c *Client) GetRun(ctx context.Context, scheduleID, runID string, _ *GetRunOptions) (ScheduleRun, error) {
	if scheduleID == "" || runID == "" {
		return ScheduleRun{}, errors.New("schedules.GetRun: scheduleID and runID are required")
	}
	body, err := c.jsonRequest(ctx, http.MethodGet,
		fmt.Sprintf("%s/schedules/%s/runs/%s", c.endpoint, scheduleID, runID),
		nil, "", http.StatusOK)
	if err != nil {
		return ScheduleRun{}, err
	}
	var out ScheduleRun
	if err := json.Unmarshal(body, &out); err != nil {
		return ScheduleRun{}, fmt.Errorf("schedules.GetRun: decode: %w", err)
	}
	return out, nil
}

// CreateOrUpdateOptions is the optional parameter set for CreateOrUpdate.
type CreateOrUpdateOptions struct {
	ClientRequestID string
}

// CreateOrUpdate creates or replaces a schedule by id.
// PUT /schedules/{id}; service returns 200 (update) or 201 (create).
func (c *Client) CreateOrUpdate(ctx context.Context, id string, schedule Schedule, opts *CreateOrUpdateOptions) (Schedule, error) {
	if id == "" {
		return Schedule{}, errors.New("schedules.CreateOrUpdate: id is required")
	}
	payload, err := json.Marshal(schedule)
	if err != nil {
		return Schedule{}, fmt.Errorf("schedules.CreateOrUpdate: marshal: %w", err)
	}
	headers := map[string]string{}
	if opts != nil && opts.ClientRequestID != "" {
		headers["x-ms-client-request-id"] = opts.ClientRequestID
	}
	body, err := c.jsonRequestWithHeaders(ctx, http.MethodPut,
		fmt.Sprintf("%s/schedules/%s", c.endpoint, id),
		payload, "application/json", headers, http.StatusOK, http.StatusCreated)
	if err != nil {
		return Schedule{}, err
	}
	var out Schedule
	if err := json.Unmarshal(body, &out); err != nil {
		return Schedule{}, fmt.Errorf("schedules.CreateOrUpdate: decode: %w", err)
	}
	return out, nil
}

// DeleteOptions is the optional parameter set for Delete.
type DeleteOptions struct{}

// DeleteResponse is empty; Delete is success/failure.
type DeleteResponse struct{}

// Delete removes a schedule by id.
// DELETE /schedules/{id} returns 204.
func (c *Client) Delete(ctx context.Context, id string, _ *DeleteOptions) (DeleteResponse, error) {
	if id == "" {
		return DeleteResponse{}, errors.New("schedules.Delete: id is required")
	}
	if _, err := c.jsonRequest(ctx, http.MethodDelete,
		fmt.Sprintf("%s/schedules/%s", c.endpoint, id),
		nil, "", http.StatusNoContent); err != nil {
		return DeleteResponse{}, err
	}
	return DeleteResponse{}, nil
}

// --- helpers ---

func (c *Client) jsonRequest(ctx context.Context, method, url string, body []byte, contentType string, okStatuses ...int) ([]byte, error) {
	return c.jsonRequestWithHeaders(ctx, method, url, body, contentType, nil, okStatuses...)
}

func (c *Client) jsonRequestWithHeaders(ctx context.Context, method, url string, body []byte, contentType string, extraHeaders map[string]string, okStatuses ...int) ([]byte, error) {
	req, err := runtime.NewRequest(ctx, method, url)
	if err != nil {
		return nil, err
	}
	req.Raw().Header.Set("foundry-features", foundryHeader)
	req.Raw().Header.Set("Accept", "application/json")
	for k, v := range extraHeaders {
		req.Raw().Header.Set(k, v)
	}
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
