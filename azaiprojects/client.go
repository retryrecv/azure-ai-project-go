package azaiprojects

import (
	"errors"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
)

// scopeAIAzureCom is the OAuth scope used when authenticating against the
// Foundry / AI Project endpoint. Matches src/aiProjectClient.ts.
const scopeAIAzureCom = "https://ai.azure.com/.default"

// Client is the entry point for the ai-projects service. It mirrors
// AIProjectClient from @azure/ai-projects.
//
// Sub-clients (Connections, Deployments, Indexes, Datasets) are constructed
// from this Client and share its endpoint, api-version, and pipeline.
type Client struct {
	endpoint   string
	apiVersion string
	cred       azcore.TokenCredential
	opts       ClientOptions
	pl         runtime.Pipeline
}

// NewClient constructs a Client targeting endpoint, authenticated with cred.
//
// Pass nil for opts to accept defaults; APIVersion defaults to APIVersionV1.
func NewClient(endpoint string, cred azcore.TokenCredential, opts *ClientOptions) (*Client, error) {
	if endpoint == "" {
		return nil, errors.New("azaiprojects: endpoint is required")
	}
	if cred == nil {
		return nil, errors.New("azaiprojects: cred is required")
	}
	if opts == nil {
		opts = &ClientOptions{}
	}
	apiVersion := opts.APIVersion
	if apiVersion == "" {
		apiVersion = APIVersionV1
	}

	bearer := runtime.NewBearerTokenPolicy(cred, []string{scopeAIAzureCom}, nil)
	pl := runtime.NewPipeline(
		ModuleName,
		ModuleVersion,
		runtime.PipelineOptions{PerRetry: []policy.Policy{bearer}},
		&opts.ClientOptions,
	)

	return &Client{
		endpoint:   endpoint,
		apiVersion: apiVersion,
		cred:       cred,
		opts:       *opts,
		pl:         pl,
	}, nil
}

// Endpoint returns the service endpoint configured on the client.
func (c *Client) Endpoint() string { return c.endpoint }

// APIVersion returns the service API version in use.
func (c *Client) APIVersion() string { return c.apiVersion }
