package schedules

import "encoding/json"

// Schedule represents a schedule resource.
//
// Trigger and Task are kept as json.RawMessage; the TriggerUnion
// (Cron|Recurrence|OneTime) and ScheduleTaskUnion are pass-through —
// caller decodes the payload type via the inner "type" field when needed.
type Schedule struct {
	ID                 string            `json:"id,omitempty"`
	DisplayName        string            `json:"displayName,omitempty"`
	Description        string            `json:"description,omitempty"`
	Enabled            bool              `json:"enabled"`
	ProvisioningStatus string            `json:"provisioningStatus,omitempty"`
	Trigger            json.RawMessage   `json:"trigger"`
	Task               json.RawMessage   `json:"task"`
	Tags               map[string]string `json:"tags,omitempty"`
	Properties         map[string]string `json:"properties,omitempty"`
	SystemData         map[string]string `json:"systemData,omitempty"`
}

// ScheduleRun represents a single execution of a schedule.
type ScheduleRun struct {
	ID          string            `json:"id"`
	ScheduleID  string            `json:"scheduleId"`
	Success     bool              `json:"success"`
	TriggerTime string            `json:"triggerTime,omitempty"`
	Error       string            `json:"error,omitempty"`
	Properties  map[string]string `json:"properties,omitempty"`
}
