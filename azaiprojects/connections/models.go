package connections

// ConnectionType is the category of a Connection. Values mirror the
// "type" discriminator returned by the service.
type ConnectionType string

const (
	ConnectionTypeAzureOpenAI         ConnectionType = "AzureOpenAI"
	ConnectionTypeAzureBlob           ConnectionType = "AzureBlob"
	ConnectionTypeAzureStorageAccount ConnectionType = "AzureStorageAccount"
	ConnectionTypeCognitiveSearch     ConnectionType = "CognitiveSearch"
	ConnectionTypeCosmosDB            ConnectionType = "CosmosDB"
	ConnectionTypeAPIKey              ConnectionType = "ApiKey"
	ConnectionTypeAppConfig           ConnectionType = "AppConfig"
	ConnectionTypeAppInsights         ConnectionType = "AppInsights"
	ConnectionTypeCustomKeys          ConnectionType = "CustomKeys"
	ConnectionTypeRemoteToolPreview   ConnectionType = "RemoteTool_Preview"
)

// CredentialType is the discriminator for BaseCredentials.
type CredentialType string

const (
	CredentialTypeAPIKey                 CredentialType = "ApiKey"
	CredentialTypeAAD                    CredentialType = "AAD"
	CredentialTypeCustomKeys             CredentialType = "CustomKeys"
	CredentialTypeSAS                    CredentialType = "SAS"
	CredentialTypeNone                   CredentialType = "None"
	CredentialTypeAgenticIdentityPreview CredentialType = "AgenticIdentityToken_Preview"
)

// BaseCredentials is a flattened view of the credential union returned by
// the service. The full discriminated split lands in a follow-up task; for
// now the union is collapsed to one struct that carries every known field.
type BaseCredentials struct {
	Type CredentialType `json:"type"`

	// Set for ApiKey credentials.
	APIKey string `json:"key,omitempty"`

	// Set for SAS credentials.
	SASToken string `json:"sasToken,omitempty"`

	// Set for CustomKeys credentials.
	Keys map[string]string `json:"keys,omitempty"`
}

// Connection mirrors @azure/ai-projects' Connection model.
type Connection struct {
	Name        string            `json:"name"`
	ID          string            `json:"id"`
	Type        ConnectionType    `json:"type"`
	Target      string            `json:"target"`
	IsDefault   bool              `json:"isDefault"`
	Credentials BaseCredentials   `json:"credentials"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}
