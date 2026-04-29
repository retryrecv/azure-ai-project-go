package evaluationtaxonomies

import (
	"encoding/json"

	"github.com/sambo/ai-projects-go/azaiprojects/internal/shared"
)

// EvaluationTaxonomy represents a taxonomy resource.
//
// TaxonomyInput and TaxonomyCategories are kept as json.RawMessage; the
// underlying TS types (EvaluationTaxonomyInputUnion with discriminator,
// TaxonomyCategory[]) are not flattened in this Go port.
type EvaluationTaxonomy struct {
	ID                 string            `json:"id,omitempty"`
	Name               string            `json:"name,omitempty"`
	Version            string            `json:"version,omitempty"`
	Description        string            `json:"description,omitempty"`
	Tags               map[string]string `json:"tags,omitempty"`
	TaxonomyInput      json.RawMessage   `json:"taxonomyInput,omitempty"`
	TaxonomyCategories json.RawMessage   `json:"taxonomyCategories,omitempty"`
	Properties         map[string]string `json:"properties,omitempty"`
}

// EvaluationTaxonomiesPage is one page returned by GET /evaluationtaxonomies.
type EvaluationTaxonomiesPage struct {
	shared.PageResponse[EvaluationTaxonomy]
}
