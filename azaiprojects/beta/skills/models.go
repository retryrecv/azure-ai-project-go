package skills

// SkillObject represents a skill resource.
type SkillObject struct {
	SkillID     string            `json:"skill_id"`
	HasBlob     bool              `json:"has_blob"`
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// CreateSkillBody is the JSON body for POST /skills.
type CreateSkillBody struct {
	Name         string            `json:"name"`
	Description  string            `json:"description,omitempty"`
	Instructions string            `json:"instructions,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

// UpdateSkillBody is the JSON body for POST /skills/{name} (update).
type UpdateSkillBody struct {
	Description  string            `json:"description,omitempty"`
	Instructions string            `json:"instructions,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

// DeleteSkillResponse is the response for DELETE /skills/{name}.
type DeleteSkillResponse struct {
	Name    string `json:"name"`
	Deleted bool   `json:"deleted"`
}

// SkillsPage is one cursor page of skills returned by GET /skills.
type SkillsPage struct {
	Data    []SkillObject `json:"data"`
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
