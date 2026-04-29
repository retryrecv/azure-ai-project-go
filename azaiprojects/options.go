package azaiprojects

import "github.com/Azure/azure-sdk-for-go/sdk/azcore"

// APIVersionV1 is the default service API version used when ClientOptions.APIVersion is empty.
const APIVersionV1 = "v1"

// ClientOptions configures the Client.
//
// It embeds azcore.ClientOptions so callers can configure transport, retry,
// telemetry, and per-call policies the same way as other azure-sdk-for-go clients.
type ClientOptions struct {
	azcore.ClientOptions

	// APIVersion overrides the service API version. Defaults to APIVersionV1.
	APIVersion string
}
