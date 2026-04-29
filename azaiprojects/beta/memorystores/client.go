package memorystores

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
	moduleName    = "azaiprojects/beta/memorystores"
	moduleVersion = "0.1.0"
	defaultScope  = "https://ai.azure.com/.default"
	defaultAPIVer = "v1"
	foundryHeader = "MemoryStores=V1Preview"
)

// Client provides the beta.memoryStores operation group.
type Client struct {
	endpoint   string
	apiVersion string
	pl         runtime.Pipeline
}

// ClientOptions configures the memorystores client.
type ClientOptions struct {
	azcore.ClientOptions
	APIVersion string
}

// NewClient constructs a memorystores client targeting endpoint.
func NewClient(endpoint string, cred azcore.TokenCredential, opts *ClientOptions) (*Client, error) {
	if endpoint == "" {
		return nil, errors.New("memorystores: endpoint is required")
	}
	if cred == nil {
		return nil, errors.New("memorystores: cred is required")
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

// NewListPager returns a Pager that issues GET /memory_stores.
func (c *Client) NewListPager(opts *ListOptions) *runtime.Pager[MemoryStoresPage] {
	first := true
	return runtime.NewPager(runtime.PagingHandler[MemoryStoresPage]{
		More: func(page MemoryStoresPage) bool {
			return page.HasMore && page.LastID != ""
		},
		Fetcher: func(ctx context.Context, page *MemoryStoresPage) (MemoryStoresPage, error) {
			req, err := runtime.NewRequest(ctx, http.MethodGet, c.endpoint+"/memory_stores")
			if err != nil {
				return MemoryStoresPage{}, err
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
				return MemoryStoresPage{}, err
			}
			var out MemoryStoresPage
			if err := json.Unmarshal(body, &out); err != nil {
				return MemoryStoresPage{}, fmt.Errorf("memorystores.List: decode: %w", err)
			}
			return out, nil
		},
	})
}

// GetOptions is the optional parameter set for Get.
type GetOptions struct{}

// Get retrieves a memory store by name.
func (c *Client) Get(ctx context.Context, name string, _ *GetOptions) (MemoryStore, error) {
	if name == "" {
		return MemoryStore{}, errors.New("memorystores.Get: name is required")
	}
	body, err := c.jsonRequest(ctx, http.MethodGet,
		fmt.Sprintf("%s/memory_stores/%s", c.endpoint, name), nil, "", http.StatusOK)
	if err != nil {
		return MemoryStore{}, err
	}
	var out MemoryStore
	if err := json.Unmarshal(body, &out); err != nil {
		return MemoryStore{}, fmt.Errorf("memorystores.Get: decode: %w", err)
	}
	return out, nil
}

// CreateOptions is the optional parameter set for Create.
type CreateOptions struct{}

// Create creates a memory store.
// POST /memory_stores returns 200.
func (c *Client) Create(ctx context.Context, body CreateMemoryStoreBody, _ *CreateOptions) (MemoryStore, error) {
	if body.Name == "" {
		return MemoryStore{}, errors.New("memorystores.Create: body.Name is required")
	}
	if len(body.Definition) == 0 {
		return MemoryStore{}, errors.New("memorystores.Create: body.Definition is required")
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return MemoryStore{}, fmt.Errorf("memorystores.Create: marshal: %w", err)
	}
	respBody, err := c.jsonRequest(ctx, http.MethodPost,
		c.endpoint+"/memory_stores", payload, "application/json", http.StatusOK)
	if err != nil {
		return MemoryStore{}, err
	}
	var out MemoryStore
	if err := json.Unmarshal(respBody, &out); err != nil {
		return MemoryStore{}, fmt.Errorf("memorystores.Create: decode: %w", err)
	}
	return out, nil
}

// UpdateOptions is the optional parameter set for Update.
type UpdateOptions struct{}

// Update updates a memory store.
// POST /memory_stores/{name} returns 200.
func (c *Client) Update(ctx context.Context, name string, body UpdateMemoryStoreBody, _ *UpdateOptions) (MemoryStore, error) {
	if name == "" {
		return MemoryStore{}, errors.New("memorystores.Update: name is required")
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return MemoryStore{}, fmt.Errorf("memorystores.Update: marshal: %w", err)
	}
	respBody, err := c.jsonRequest(ctx, http.MethodPost,
		fmt.Sprintf("%s/memory_stores/%s", c.endpoint, name), payload, "application/json", http.StatusOK)
	if err != nil {
		return MemoryStore{}, err
	}
	var out MemoryStore
	if err := json.Unmarshal(respBody, &out); err != nil {
		return MemoryStore{}, fmt.Errorf("memorystores.Update: decode: %w", err)
	}
	return out, nil
}

// DeleteOptions is the optional parameter set for Delete.
type DeleteOptions struct{}

// Delete deletes a memory store by name.
// DELETE /memory_stores/{name} returns 200 with body {object, name, deleted}.
func (c *Client) Delete(ctx context.Context, name string, _ *DeleteOptions) (DeleteMemoryStoreResponse, error) {
	if name == "" {
		return DeleteMemoryStoreResponse{}, errors.New("memorystores.Delete: name is required")
	}
	respBody, err := c.jsonRequest(ctx, http.MethodDelete,
		fmt.Sprintf("%s/memory_stores/%s", c.endpoint, name), nil, "", http.StatusOK)
	if err != nil {
		return DeleteMemoryStoreResponse{}, err
	}
	var out DeleteMemoryStoreResponse
	if err := json.Unmarshal(respBody, &out); err != nil {
		return DeleteMemoryStoreResponse{}, fmt.Errorf("memorystores.Delete: decode: %w", err)
	}
	return out, nil
}

// DeleteScopeOptions is the optional parameter set for DeleteScope.
type DeleteScopeOptions struct{}

// DeleteScope deletes all memories under the given scope.
// POST /memory_stores/{name}:delete_scope returns 200.
func (c *Client) DeleteScope(ctx context.Context, name, scope string, _ *DeleteScopeOptions) (MemoryStoreDeleteScopeResponse, error) {
	if name == "" {
		return MemoryStoreDeleteScopeResponse{}, errors.New("memorystores.DeleteScope: name is required")
	}
	if scope == "" {
		return MemoryStoreDeleteScopeResponse{}, errors.New("memorystores.DeleteScope: scope is required")
	}
	payload, err := json.Marshal(struct {
		Scope string `json:"scope"`
	}{Scope: scope})
	if err != nil {
		return MemoryStoreDeleteScopeResponse{}, fmt.Errorf("memorystores.DeleteScope: marshal: %w", err)
	}
	respBody, err := c.jsonRequest(ctx, http.MethodPost,
		fmt.Sprintf("%s/memory_stores/%s:delete_scope", c.endpoint, name), payload, "application/json", http.StatusOK)
	if err != nil {
		return MemoryStoreDeleteScopeResponse{}, err
	}
	var out MemoryStoreDeleteScopeResponse
	if err := json.Unmarshal(respBody, &out); err != nil {
		return MemoryStoreDeleteScopeResponse{}, fmt.Errorf("memorystores.DeleteScope: decode: %w", err)
	}
	return out, nil
}

// UpdateMemoriesOptions is the optional parameter set for UpdateMemories.
type UpdateMemoriesOptions struct{}

// UpdateMemories starts (or supersedes) a memory update operation.
// POST /memory_stores/{name}:update_memories accepts 200/201/202.
//
// Note: the TS surface wraps this in a long-running poller. This Go port
// returns the immediate response; callers can poll GetUpdateResult by
// update_id until the status reaches a terminal value.
func (c *Client) UpdateMemories(ctx context.Context, name string, body UpdateMemoriesBody, _ *UpdateMemoriesOptions) (MemoryStoreUpdateResponse, error) {
	if name == "" {
		return MemoryStoreUpdateResponse{}, errors.New("memorystores.UpdateMemories: name is required")
	}
	if body.Scope == "" {
		return MemoryStoreUpdateResponse{}, errors.New("memorystores.UpdateMemories: body.Scope is required")
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return MemoryStoreUpdateResponse{}, fmt.Errorf("memorystores.UpdateMemories: marshal: %w", err)
	}
	respBody, err := c.jsonRequest(ctx, http.MethodPost,
		fmt.Sprintf("%s/memory_stores/%s:update_memories", c.endpoint, name), payload, "application/json",
		http.StatusOK, http.StatusCreated, http.StatusAccepted)
	if err != nil {
		return MemoryStoreUpdateResponse{}, err
	}
	if len(respBody) == 0 {
		return MemoryStoreUpdateResponse{}, nil
	}
	var out MemoryStoreUpdateResponse
	if err := json.Unmarshal(respBody, &out); err != nil {
		return MemoryStoreUpdateResponse{}, fmt.Errorf("memorystores.UpdateMemories: decode: %w", err)
	}
	return out, nil
}

// GetUpdateResultOptions is the optional parameter set for GetUpdateResult.
type GetUpdateResultOptions struct{}

// GetUpdateResult fetches the status/result of a memory update operation.
// GET /memory_stores/{name}/updates/{update_id} returns 200.
func (c *Client) GetUpdateResult(ctx context.Context, name, updateID string, _ *GetUpdateResultOptions) (MemoryStoreUpdateResponse, error) {
	if name == "" {
		return MemoryStoreUpdateResponse{}, errors.New("memorystores.GetUpdateResult: name is required")
	}
	if updateID == "" {
		return MemoryStoreUpdateResponse{}, errors.New("memorystores.GetUpdateResult: updateID is required")
	}
	respBody, err := c.jsonRequest(ctx, http.MethodGet,
		fmt.Sprintf("%s/memory_stores/%s/updates/%s", c.endpoint, name, updateID), nil, "", http.StatusOK)
	if err != nil {
		return MemoryStoreUpdateResponse{}, err
	}
	var out MemoryStoreUpdateResponse
	if err := json.Unmarshal(respBody, &out); err != nil {
		return MemoryStoreUpdateResponse{}, fmt.Errorf("memorystores.GetUpdateResult: decode: %w", err)
	}
	return out, nil
}

// SearchMemoriesOptions is the optional parameter set for SearchMemories.
type SearchMemoriesOptions struct{}

// SearchMemories searches relevant memories for a scope and conversation.
// POST /memory_stores/{name}:search_memories returns 200.
func (c *Client) SearchMemories(ctx context.Context, name string, body SearchMemoriesBody, _ *SearchMemoriesOptions) (MemoryStoreSearchResponse, error) {
	if name == "" {
		return MemoryStoreSearchResponse{}, errors.New("memorystores.SearchMemories: name is required")
	}
	if body.Scope == "" {
		return MemoryStoreSearchResponse{}, errors.New("memorystores.SearchMemories: body.Scope is required")
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return MemoryStoreSearchResponse{}, fmt.Errorf("memorystores.SearchMemories: marshal: %w", err)
	}
	respBody, err := c.jsonRequest(ctx, http.MethodPost,
		fmt.Sprintf("%s/memory_stores/%s:search_memories", c.endpoint, name), payload, "application/json", http.StatusOK)
	if err != nil {
		return MemoryStoreSearchResponse{}, err
	}
	var out MemoryStoreSearchResponse
	if err := json.Unmarshal(respBody, &out); err != nil {
		return MemoryStoreSearchResponse{}, fmt.Errorf("memorystores.SearchMemories: decode: %w", err)
	}
	return out, nil
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
