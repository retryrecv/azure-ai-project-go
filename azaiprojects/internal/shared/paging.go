// Package shared holds helpers used by sub-clients of azaiprojects.
//
// Nothing here is part of the public surface; it sits under internal/ on
// purpose so the import path stays stable as the wire format evolves.
package shared

// PageResponse is the canonical wire shape returned by every list endpoint
// in the ai-projects service: a `value` array of T plus an optional
// `nextLink` continuation token.
type PageResponse[T any] struct {
	Value    []T     `json:"value"`
	NextLink *string `json:"nextLink,omitempty"`
}
