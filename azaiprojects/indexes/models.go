package indexes

// IndexType discriminates the Index union.
type IndexType string

const (
	IndexTypeAzureSearch        IndexType = "AzureSearch"
	IndexTypeManagedAzureSearch IndexType = "ManagedAzureSearch"
	IndexTypeCosmosDBNoSql      IndexType = "CosmosDBNoSqlVectorStore"
)

// Index is the base view shared by every concrete index type.
type Index struct {
	Type        IndexType         `json:"type"`
	ID          string            `json:"id,omitempty"`
	Name        string            `json:"name"`
	Version     string            `json:"version"`
	Description string            `json:"description,omitempty"`
	Tags        map[string]string `json:"tags,omitempty"`
}

// FieldMapping describes how index fields map to source document fields.
type FieldMapping struct {
	ContentFields  []string `json:"contentFields"`
	FilepathField  string   `json:"filepathField,omitempty"`
	TitleField     string   `json:"titleField,omitempty"`
	URLField       string   `json:"urlField,omitempty"`
	VectorFields   []string `json:"vectorFields,omitempty"`
	MetadataFields []string `json:"metadataFields,omitempty"`
}

// EmbeddingConfiguration is referenced by CosmosDBIndex.
type EmbeddingConfiguration struct {
	ModelDeploymentName string `json:"modelDeploymentName"`
	EmbeddingField      string `json:"embeddingField"`
}

// AzureAISearchIndex points to an existing index in an Azure AI Search resource.
type AzureAISearchIndex struct {
	Index
	ConnectionName string        `json:"connectionName"`
	IndexName      string        `json:"indexName"`
	FieldMapping   *FieldMapping `json:"fieldMapping,omitempty"`
}

// ManagedAzureAISearchIndex points to a managed vector store.
type ManagedAzureAISearchIndex struct {
	Index
	VectorStoreID string `json:"vectorStoreId"`
}

// CosmosDBIndex points to a CosmosDB-backed vector store.
type CosmosDBIndex struct {
	Index
	ConnectionName         string                 `json:"connectionName"`
	DatabaseName           string                 `json:"databaseName"`
	ContainerName          string                 `json:"containerName"`
	EmbeddingConfiguration EmbeddingConfiguration `json:"embeddingConfiguration"`
	FieldMapping           FieldMapping           `json:"fieldMapping"`
}
