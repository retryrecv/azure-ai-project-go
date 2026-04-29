package connections

import (
	"encoding/json"
	"testing"
)

func TestConnection_RoundTripAzureOpenAI(t *testing.T) {
	const body = `{
		"name":"c1",
		"id":"/subscriptions/.../c1",
		"type":"AzureOpenAI",
		"target":"https://example.openai.azure.com",
		"isDefault":true,
		"credentials":{"type":"ApiKey","key":"k"},
		"metadata":{"k":"v"}
	}`

	var got Connection
	if err := json.Unmarshal([]byte(body), &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if got.Name != "c1" {
		t.Errorf("Name = %q, want c1", got.Name)
	}
	if got.Type != ConnectionTypeAzureOpenAI {
		t.Errorf("Type = %q, want %q", got.Type, ConnectionTypeAzureOpenAI)
	}
	if !got.IsDefault {
		t.Error("IsDefault = false, want true")
	}
	if got.Credentials.Type != CredentialTypeAPIKey {
		t.Errorf("Credentials.Type = %q, want %q", got.Credentials.Type, CredentialTypeAPIKey)
	}
	if got.Credentials.APIKey != "k" {
		t.Errorf("Credentials.APIKey = %q, want k", got.Credentials.APIKey)
	}
	if got.Metadata["k"] != "v" {
		t.Errorf("Metadata[k] = %q, want v", got.Metadata["k"])
	}

	round, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var rt Connection
	if err := json.Unmarshal(round, &rt); err != nil {
		t.Fatalf("re-Unmarshal: %v", err)
	}
	if rt.Type != ConnectionTypeAzureOpenAI || rt.Credentials.APIKey != "k" {
		t.Errorf("round-trip lost fields: %+v", rt)
	}
}

func TestConnectionTypeConstants(t *testing.T) {
	cases := map[ConnectionType]string{
		ConnectionTypeAzureOpenAI:         "AzureOpenAI",
		ConnectionTypeAzureBlob:           "AzureBlob",
		ConnectionTypeAzureStorageAccount: "AzureStorageAccount",
		ConnectionTypeCognitiveSearch:     "CognitiveSearch",
		ConnectionTypeCosmosDB:            "CosmosDB",
		ConnectionTypeAPIKey:              "ApiKey",
		ConnectionTypeAppConfig:           "AppConfig",
		ConnectionTypeAppInsights:         "AppInsights",
		ConnectionTypeCustomKeys:          "CustomKeys",
		ConnectionTypeRemoteToolPreview:   "RemoteTool_Preview",
	}
	for got, want := range cases {
		if string(got) != want {
			t.Errorf("ConnectionType %q != wire value %q", got, want)
		}
	}
}
