package memorystores

import (
	"encoding/json"

	"github.com/sambo/ai-projects-go/azaiprojects/internal/shared"
)

// MemoryStore represents a memory store resource.
//
// Definition is left as json.RawMessage to pass through the
// MemoryStoreDefinitionUnion discriminator without flattening.
type MemoryStore struct {
	Object      string            `json:"object,omitempty"`
	Name        string            `json:"name,omitempty"`
	Description string            `json:"description,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	CreatedAt   shared.UnixSeconds `json:"created_at,omitempty"`
	UpdatedAt   shared.UnixSeconds `json:"updated_at,omitempty"`
	Definition  json.RawMessage   `json:"definition,omitempty"`
	// ID mirrors the wire field "id".
	ID string `json:"id,omitempty"`
}

// MemoryStoresPage is one cursor page returned by GET /memory_stores.
type MemoryStoresPage struct {
	Data    []MemoryStore `json:"data"`
	FirstID string        `json:"first_id,omitempty"`
	LastID  string        `json:"last_id,omitempty"`
	HasMore bool          `json:"has_more"`
}

// PageOrder is the order parameter for cursor-paged list operations.
type PageOrder string

const (
	PageOrderAsc  PageOrder = "asc"
	PageOrderDesc PageOrder = "desc"
)

// CreateMemoryStoreBody is the JSON body for POST /memory_stores.
type CreateMemoryStoreBody struct {
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	Definition  json.RawMessage   `json:"definition"`
}

// UpdateMemoryStoreBody is the JSON body for POST /memory_stores/{name}.
type UpdateMemoryStoreBody struct {
	Description string            `json:"description,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// DeleteMemoryStoreResponse is the response for DELETE /memory_stores/{name}.
type DeleteMemoryStoreResponse struct {
	Object  string `json:"object,omitempty"`
	Name    string `json:"name,omitempty"`
	Deleted bool   `json:"deleted"`
}

// MemoryStoreDeleteScopeResponse is the response for POST /memory_stores/{name}:delete_scope.
type MemoryStoreDeleteScopeResponse struct {
	Object  string `json:"object,omitempty"`
	Name    string `json:"name,omitempty"`
	Scope   string `json:"scope,omitempty"`
	Deleted bool   `json:"deleted"`
}

// UpdateMemoriesBody is the JSON body for POST /memory_stores/{name}:update_memories.
//
// Items are kept as []json.RawMessage; the underlying TS type is
// Record<string, unknown>[] (free-form messages).
type UpdateMemoriesBody struct {
	Scope             string            `json:"scope"`
	Items             []json.RawMessage `json:"items,omitempty"`
	PreviousUpdateID  string            `json:"previous_update_id,omitempty"`
	UpdateDelayInSecs *int32            `json:"update_delay,omitempty"`
}

// SearchMemoriesBody is the JSON body for POST /memory_stores/{name}:search_memories.
type SearchMemoriesBody struct {
	Scope            string            `json:"scope"`
	Items            []json.RawMessage `json:"items,omitempty"`
	PreviousSearchID string            `json:"previous_search_id,omitempty"`
	// Options is the MemorySearchOptions union pass-through.
	Options json.RawMessage `json:"options,omitempty"`
}

// MemoryStoreUpdateResponse is the response of GET /memory_stores/{name}/updates/{update_id}
// and the immediate response of POST :update_memories.
//
// Result and Error are pass-throughs.
type MemoryStoreUpdateResponse struct {
	UpdateID     string          `json:"update_id,omitempty"`
	Status       string          `json:"status,omitempty"`
	SupersededBy string          `json:"superseded_by,omitempty"`
	Result       json.RawMessage `json:"result,omitempty"`
	Error        json.RawMessage `json:"error,omitempty"`
}

// MemoryStoreSearchResponse is the response of POST :search_memories.
//
// Memories and Usage are pass-throughs.
type MemoryStoreSearchResponse struct {
	SearchID string          `json:"search_id,omitempty"`
	Memories json.RawMessage `json:"memories,omitempty"`
	Usage    json.RawMessage `json:"usage,omitempty"`
}
