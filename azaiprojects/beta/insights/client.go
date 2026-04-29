package insights

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
	moduleName    = "azaiprojects/beta/insights"
	moduleVersion = "0.1.0"
	defaultScope  = "https://ai.azure.com/.default"
	defaultAPIVer = "v1"
	foundryHeader = "Insights=V1Preview"
)

// Client provides the beta.insights operation group.
type Client struct {
	endpoint   string
	apiVersion string
	pl         runtime.Pipeline
}

// ClientOptions configures the insights client.
type ClientOptions struct {
	azcore.ClientOptions
	APIVersion string
}

// NewClient constructs an insights client targeting endpoint.
func NewClient(endpoint string, cred azcore.TokenCredential, opts *ClientOptions) (*Client, error) {
	if endpoint == "" {
		return nil, errors.New("insights: endpoint is required")
	}
	if cred == nil {
		return nil, errors.New("insights: cred is required")
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
	InsightType        *string
	EvalID             *string
	RunID              *string
	AgentName          *string
	IncludeCoordinates *bool
	ClientRequestID    string
}

// NewListPager returns a Pager that issues GET /insights.
func (c *Client) NewListPager(opts *ListOptions) *runtime.Pager[InsightsPage] {
	first := true
	return runtime.NewPager(runtime.PagingHandler[InsightsPage]{
		More: func(page InsightsPage) bool {
			return page.NextLink != nil && *page.NextLink != ""
		},
		Fetcher: func(ctx context.Context, page *InsightsPage) (InsightsPage, error) {
			var url string
			if first || page == nil || page.NextLink == nil {
				first = false
				url = c.endpoint + "/insights"
			} else {
				url = *page.NextLink
			}
			req, err := runtime.NewRequest(ctx, http.MethodGet, url)
			if err != nil {
				return InsightsPage{}, err
			}
			req.Raw().Header.Set("foundry-features", foundryHeader)
			req.Raw().Header.Set("Accept", "application/json")
			if opts != nil && opts.ClientRequestID != "" {
				req.Raw().Header.Set("x-ms-client-request-id", opts.ClientRequestID)
			}
			q := req.Raw().URL.Query()
			if opts != nil {
				if opts.InsightType != nil {
					q.Set("type", *opts.InsightType)
				}
				if opts.EvalID != nil {
					q.Set("evalId", *opts.EvalID)
				}
				if opts.RunID != nil {
					q.Set("runId", *opts.RunID)
				}
				if opts.AgentName != nil {
					q.Set("agentName", *opts.AgentName)
				}
				if opts.IncludeCoordinates != nil {
					q.Set("includeCoordinates", strconv.FormatBool(*opts.IncludeCoordinates))
				}
			}
			q.Set("api-version", c.apiVersion)
			req.Raw().URL.RawQuery = q.Encode()
			body, err := c.do(req, http.StatusOK)
			if err != nil {
				return InsightsPage{}, err
			}
			var out InsightsPage
			if err := json.Unmarshal(body, &out); err != nil {
				return InsightsPage{}, fmt.Errorf("insights.List: decode: %w", err)
			}
			return out, nil
		},
	})
}

// GetOptions is the optional parameter set for Get.
type GetOptions struct {
	IncludeCoordinates *bool
	ClientRequestID    string
}

// Get retrieves an insight by ID.
func (c *Client) Get(ctx context.Context, insightID string, opts *GetOptions) (Insight, error) {
	if insightID == "" {
		return Insight{}, errors.New("insights.Get: insightID is required")
	}
	req, err := runtime.NewRequest(ctx, http.MethodGet,
		fmt.Sprintf("%s/insights/%s", c.endpoint, insightID))
	if err != nil {
		return Insight{}, err
	}
	req.Raw().Header.Set("foundry-features", foundryHeader)
	req.Raw().Header.Set("Accept", "application/json")
	if opts != nil && opts.ClientRequestID != "" {
		req.Raw().Header.Set("x-ms-client-request-id", opts.ClientRequestID)
	}
	q := req.Raw().URL.Query()
	q.Set("api-version", c.apiVersion)
	if opts != nil && opts.IncludeCoordinates != nil {
		q.Set("includeCoordinates", strconv.FormatBool(*opts.IncludeCoordinates))
	}
	req.Raw().URL.RawQuery = q.Encode()
	body, err := c.do(req, http.StatusOK)
	if err != nil {
		return Insight{}, err
	}
	var out Insight
	if err := json.Unmarshal(body, &out); err != nil {
		return Insight{}, fmt.Errorf("insights.Get: decode: %w", err)
	}
	return out, nil
}

// GenerateOptions is the optional parameter set for Generate.
type GenerateOptions struct {
	RepeatabilityRequestID string
	RepeatabilityFirstSent string
}

// Generate generates a new insight.
// POST /insights returns 201.
func (c *Client) Generate(ctx context.Context, insight Insight, opts *GenerateOptions) (Insight, error) {
	payload, err := json.Marshal(insight)
	if err != nil {
		return Insight{}, fmt.Errorf("insights.Generate: marshal: %w", err)
	}
	req, err := runtime.NewRequest(ctx, http.MethodPost, c.endpoint+"/insights")
	if err != nil {
		return Insight{}, err
	}
	req.Raw().Header.Set("foundry-features", foundryHeader)
	req.Raw().Header.Set("Accept", "application/json")
	if opts != nil {
		if opts.RepeatabilityRequestID != "" {
			req.Raw().Header.Set("repeatability-request-id", opts.RepeatabilityRequestID)
		}
		if opts.RepeatabilityFirstSent != "" {
			req.Raw().Header.Set("repeatability-first-sent", opts.RepeatabilityFirstSent)
		}
	}
	q := req.Raw().URL.Query()
	q.Set("api-version", c.apiVersion)
	req.Raw().URL.RawQuery = q.Encode()
	if err := req.SetBody(byteSeeker{bytes.NewReader(payload)}, "application/json"); err != nil {
		return Insight{}, err
	}
	body, err := c.do(req, http.StatusCreated)
	if err != nil {
		return Insight{}, err
	}
	var out Insight
	if err := json.Unmarshal(body, &out); err != nil {
		return Insight{}, fmt.Errorf("insights.Generate: decode: %w", err)
	}
	return out, nil
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
