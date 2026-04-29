package shared

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestPageResponse_Unmarshal(t *testing.T) {
	const body = `{"value":[1,2,3],"nextLink":"x"}`
	var got PageResponse[int]
	if err := json.Unmarshal([]byte(body), &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if !reflect.DeepEqual(got.Value, []int{1, 2, 3}) {
		t.Errorf("Value = %v, want [1 2 3]", got.Value)
	}
	if got.NextLink == nil || *got.NextLink != "x" {
		t.Errorf("NextLink = %v, want pointer to \"x\"", got.NextLink)
	}
}

func TestPageResponse_OmitEmptyNextLink(t *testing.T) {
	const body = `{"value":[]}`
	var got PageResponse[int]
	if err := json.Unmarshal([]byte(body), &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got.NextLink != nil {
		t.Errorf("NextLink = %v, want nil when omitted", got.NextLink)
	}
}
