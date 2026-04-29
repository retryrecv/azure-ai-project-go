package azaiprojects

import "testing"

func TestClientOptions_ZeroValueAPIVersionIsEmpty(t *testing.T) {
	var opts ClientOptions
	if opts.APIVersion != "" {
		t.Fatalf("zero-value ClientOptions.APIVersion = %q, want empty (defaulting handled by NewClient)", opts.APIVersion)
	}
}

func TestAPIVersionV1Constant(t *testing.T) {
	if APIVersionV1 != "v1" {
		t.Fatalf("APIVersionV1 = %q, want %q", APIVersionV1, "v1")
	}
}
