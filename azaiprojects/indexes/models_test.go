package indexes

import (
	"encoding/json"
	"testing"
)

func TestAzureAISearchIndex_RoundTrip(t *testing.T) {
	const body = `{
		"type":"AzureSearch",
		"name":"my-azure-search-index",
		"version":"1.0",
		"connectionName":"sc",
		"indexName":"docs"
	}`
	var got AzureAISearchIndex
	if err := json.Unmarshal([]byte(body), &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got.Type != IndexTypeAzureSearch {
		t.Errorf("Type = %q, want %q", got.Type, IndexTypeAzureSearch)
	}
	if got.IndexName != "docs" || got.ConnectionName != "sc" {
		t.Errorf("got = %+v", got)
	}
}

func TestIndexTypeWireValues(t *testing.T) {
	cases := map[IndexType]string{
		IndexTypeAzureSearch:        "AzureSearch",
		IndexTypeManagedAzureSearch: "ManagedAzureSearch",
		IndexTypeCosmosDBNoSql:      "CosmosDBNoSqlVectorStore",
	}
	for got, want := range cases {
		if string(got) != want {
			t.Errorf("%q != wire value %q", got, want)
		}
	}
}
