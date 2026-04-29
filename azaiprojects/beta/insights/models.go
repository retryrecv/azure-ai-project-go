package insights

import (
	"encoding/json"
	"time"

	"github.com/sambo/ai-projects-go/azaiprojects/internal/shared"
)

// Insight represents an insights report.
//
// Request and Result are kept as json.RawMessage; the underlying TS unions
// (InsightRequestUnion / InsightResultUnion with discriminators
// EvaluationRunClusterInsight, AgentClusterInsight, EvaluationComparison)
// are not flattened in this Go port.
type Insight struct {
	InsightID   string           `json:"id,omitempty"`
	Metadata    *InsightMetadata `json:"metadata,omitempty"`
	State       string           `json:"state,omitempty"`
	DisplayName string           `json:"displayName,omitempty"`
	Request     json.RawMessage  `json:"request,omitempty"`
	Result      json.RawMessage  `json:"result,omitempty"`
}

// InsightMetadata is the metadata block on an Insight.
type InsightMetadata struct {
	CreatedAt   time.Time  `json:"createdAt,omitempty"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`
}

// InsightsPage is one page returned by GET /insights (link pagination via nextLink).
type InsightsPage struct {
	shared.PageResponse[Insight]
}
