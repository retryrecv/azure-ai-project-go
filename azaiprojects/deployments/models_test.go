package deployments

import (
	"encoding/json"
	"testing"
)

func TestModelDeployment_RoundTrip(t *testing.T) {
	const body = `{
		"type":"ModelDeployment",
		"name":"my-gpt4o",
		"modelName":"gpt-4o",
		"modelVersion":"2024-08-06",
		"modelPublisher":"openai",
		"sku":{"capacity":10,"family":"f","name":"s1","size":"sm","tier":"Standard"}
	}`
	var got ModelDeployment
	if err := json.Unmarshal([]byte(body), &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got.ModelName != "gpt-4o" {
		t.Errorf("ModelName = %q, want gpt-4o", got.ModelName)
	}
	if got.Type != DeploymentTypeModel {
		t.Errorf("Type = %q, want %q", got.Type, DeploymentTypeModel)
	}
	if got.Sku.Capacity != 10 {
		t.Errorf("Sku.Capacity = %d, want 10", got.Sku.Capacity)
	}
}
