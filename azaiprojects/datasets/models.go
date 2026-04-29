package datasets

// DatasetType discriminates the DatasetVersion union.
type DatasetType string

const (
	DatasetTypeURIFile   DatasetType = "uri_file"
	DatasetTypeURIFolder DatasetType = "uri_folder"
)

// PendingUploadType is the kind of pending upload requested.
type PendingUploadType string

const (
	PendingUploadTypeNone          PendingUploadType = "None"
	PendingUploadTypeBlobReference PendingUploadType = "BlobReference"
)

// DatasetVersion is the base view shared by FileDatasetVersion and
// FolderDatasetVersion. The Type field discriminates the union.
type DatasetVersion struct {
	DataURI        string            `json:"dataUri"`
	Type           DatasetType       `json:"type"`
	IsReference    *bool             `json:"isReference,omitempty"`
	ConnectionName string            `json:"connectionName,omitempty"`
	ID             string            `json:"id,omitempty"`
	Name           string            `json:"name"`
	Version        string            `json:"version"`
	Description    string            `json:"description,omitempty"`
	Tags           map[string]string `json:"tags,omitempty"`
}

// FileDatasetVersion is a dataset that points to a single URI file.
type FileDatasetVersion struct {
	DatasetVersion
}

// FolderDatasetVersion is a dataset that points to a URI folder.
type FolderDatasetVersion struct {
	DatasetVersion
}

// PendingUploadRequest starts (or resumes) a pending upload for a dataset version.
type PendingUploadRequest struct {
	PendingUploadID   string            `json:"pendingUploadId,omitempty"`
	ConnectionName    string            `json:"connectionName,omitempty"`
	PendingUploadType PendingUploadType `json:"pendingUploadType"`
}

// SasCredential is the SAS credential returned with a BlobReference.
type SasCredential struct {
	SASURI string `json:"sasUri"`
	Type   string `json:"type"`
}

// BlobReference is the storage location returned by PendingUpload and GetCredentials.
type BlobReference struct {
	BlobURI             string        `json:"blobUri"`
	StorageAccountARMID string        `json:"storageAccountArmId"`
	Credential          SasCredential `json:"credential"`
}

// PendingUploadResponse is the result of starting a pending upload.
type PendingUploadResponse struct {
	BlobReference     BlobReference     `json:"blobReference"`
	PendingUploadID   string            `json:"pendingUploadId"`
	Version           string            `json:"version,omitempty"`
	PendingUploadType PendingUploadType `json:"pendingUploadType"`
}

// DatasetCredential is returned by GetCredentials.
type DatasetCredential struct {
	BlobReference BlobReference `json:"blobReference"`
}
