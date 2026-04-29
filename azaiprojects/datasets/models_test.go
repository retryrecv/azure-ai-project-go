package datasets

import (
	"encoding/json"
	"testing"
)

func TestFileDatasetVersion_RoundTrip(t *testing.T) {
	const body = `{
		"name":"d1",
		"version":"1.0",
		"type":"uri_file",
		"dataUri":"https://example/blob.txt",
		"connectionName":"sc"
	}`
	var got FileDatasetVersion
	if err := json.Unmarshal([]byte(body), &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got.Type != DatasetTypeURIFile {
		t.Errorf("Type = %q, want %q", got.Type, DatasetTypeURIFile)
	}
	if got.DataURI != "https://example/blob.txt" || got.ConnectionName != "sc" {
		t.Errorf("got = %+v", got)
	}
}

func TestPendingUploadResponse_RoundTrip(t *testing.T) {
	const body = `{
		"pendingUploadId":"u1",
		"pendingUploadType":"BlobReference",
		"blobReference":{
			"blobUri":"https://blob/x",
			"storageAccountArmId":"/subs/.../sa",
			"credential":{"sasUri":"https://blob/x?sv=...","type":"SAS"}
		}
	}`
	var got PendingUploadResponse
	if err := json.Unmarshal([]byte(body), &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got.PendingUploadType != PendingUploadTypeBlobReference {
		t.Errorf("PendingUploadType = %q, want BlobReference", got.PendingUploadType)
	}
	if got.BlobReference.BlobURI != "https://blob/x" {
		t.Errorf("BlobURI = %q, want https://blob/x", got.BlobReference.BlobURI)
	}
	if got.BlobReference.Credential.SASURI != "https://blob/x?sv=..." {
		t.Errorf("Credential.SASURI = %q", got.BlobReference.Credential.SASURI)
	}
}
