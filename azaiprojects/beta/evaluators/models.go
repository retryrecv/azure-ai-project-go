package evaluators

import (
	"encoding/json"

	"github.com/sambo/ai-projects-go/azaiprojects/internal/shared"
)

// EvaluatorVersion represents a single evaluator version.
//
// Definition is kept as json.RawMessage; the underlying TS union
// (EvaluatorDefinitionUnion: code | prompt) is not flattened.
type EvaluatorVersion struct {
	DisplayName   string            `json:"display_name,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
	EvaluatorType string            `json:"evaluator_type,omitempty"`
	Categories    []string          `json:"categories,omitempty"`
	Definition    json.RawMessage   `json:"definition,omitempty"`
	CreatedBy     string            `json:"created_by,omitempty"`
	CreatedAt     string            `json:"created_at,omitempty"`
	ModifiedAt    string            `json:"modified_at,omitempty"`
	ID            string            `json:"id,omitempty"`
	Name          string            `json:"name,omitempty"`
	Version       string            `json:"version,omitempty"`
	Description   string            `json:"description,omitempty"`
	Tags          map[string]string `json:"tags,omitempty"`
}

// EvaluatorVersionsPage is one page of EvaluatorVersion results.
type EvaluatorVersionsPage struct {
	shared.PageResponse[EvaluatorVersion]
}

// EvaluatorType values.
const (
	EvaluatorTypeBuiltin = "builtin"
	EvaluatorTypeCustom  = "custom"
)
