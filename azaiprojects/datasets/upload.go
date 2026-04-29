package datasets

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
)

// UploadOptions controls UploadFile and UploadFolder.
type UploadOptions struct {
	// ConnectionName names the storage connection used for the pending upload.
	// If empty, the service picks the project's default storage.
	ConnectionName string
	// FilePattern, when non-nil, restricts UploadFolder to files whose full
	// path matches the regex. Ignored by UploadFile.
	FilePattern *regexp.Regexp
}

// UploadFile uploads a single local file as a uri_file dataset version.
func (c *Client) UploadFile(ctx context.Context, name, version, filePath string, opts *UploadOptions) (CreateOrUpdateResponse, error) {
	if name == "" || version == "" {
		return CreateOrUpdateResponse{}, errors.New("datasets.UploadFile: name and version are required")
	}
	info, err := os.Stat(filePath)
	if err != nil {
		return CreateOrUpdateResponse{}, fmt.Errorf("datasets.UploadFile: stat: %w", err)
	}
	if info.IsDir() {
		return CreateOrUpdateResponse{}, fmt.Errorf("datasets.UploadFile: %q is a directory; use UploadFolder", filePath)
	}

	containerSAS, blobBase, err := c.startBlobUpload(ctx, name, version, opts)
	if err != nil {
		return CreateOrUpdateResponse{}, err
	}
	blobName := filepath.Base(filePath)
	if err := uploadOneFile(ctx, containerSAS, filePath, blobName); err != nil {
		return CreateOrUpdateResponse{}, err
	}

	return c.CreateOrUpdate(ctx, name, version, FileDatasetVersion{
		DatasetVersion: DatasetVersion{
			Name:    name,
			Version: version,
			Type:    DatasetTypeURIFile,
			DataURI: blobBase + "/" + blobName,
		},
	}, nil)
}

// UploadFolder recursively uploads folderPath and creates a uri_folder dataset version.
func (c *Client) UploadFolder(ctx context.Context, name, version, folderPath string, opts *UploadOptions) (CreateOrUpdateResponse, error) {
	if name == "" || version == "" {
		return CreateOrUpdateResponse{}, errors.New("datasets.UploadFolder: name and version are required")
	}
	info, err := os.Stat(folderPath)
	if err != nil {
		return CreateOrUpdateResponse{}, fmt.Errorf("datasets.UploadFolder: stat: %w", err)
	}
	if !info.IsDir() {
		return CreateOrUpdateResponse{}, fmt.Errorf("datasets.UploadFolder: %q is a file; use UploadFile", folderPath)
	}

	var pattern *regexp.Regexp
	if opts != nil {
		pattern = opts.FilePattern
	}
	files, err := walkFiles(folderPath, pattern)
	if err != nil {
		return CreateOrUpdateResponse{}, err
	}
	if len(files) == 0 {
		return CreateOrUpdateResponse{}, errors.New("datasets.UploadFolder: folder is empty (after filtering)")
	}

	containerSAS, blobBase, err := c.startBlobUpload(ctx, name, version, opts)
	if err != nil {
		return CreateOrUpdateResponse{}, err
	}
	for _, f := range files {
		rel, err := filepath.Rel(folderPath, f)
		if err != nil {
			return CreateOrUpdateResponse{}, err
		}
		blobName := filepath.ToSlash(rel)
		if err := uploadOneFile(ctx, containerSAS, f, blobName); err != nil {
			return CreateOrUpdateResponse{}, err
		}
	}

	return c.CreateOrUpdate(ctx, name, version, FolderDatasetVersion{
		DatasetVersion: DatasetVersion{
			Name:    name,
			Version: version,
			Type:    DatasetTypeURIFolder,
			DataURI: blobBase,
		},
	}, nil)
}

// startBlobUpload calls PendingUpload and returns (containerSASURL, blobBaseURI).
func (c *Client) startBlobUpload(ctx context.Context, name, version string, opts *UploadOptions) (string, string, error) {
	req := PendingUploadRequest{PendingUploadType: PendingUploadTypeBlobReference}
	if opts != nil {
		req.ConnectionName = opts.ConnectionName
	}
	resp, err := c.PendingUpload(ctx, name, version, req, nil)
	if err != nil {
		return "", "", fmt.Errorf("datasets: pending upload: %w", err)
	}
	if resp.BlobReference.Credential.Type != "SAS" {
		return "", "", fmt.Errorf("datasets: expected SAS credential, got %q", resp.BlobReference.Credential.Type)
	}
	if resp.BlobReference.Credential.SASURI == "" {
		return "", "", errors.New("datasets: empty SAS URI in pending upload response")
	}
	if resp.BlobReference.BlobURI == "" {
		return "", "", errors.New("datasets: empty blob URI in pending upload response")
	}
	return resp.BlobReference.Credential.SASURI, strings.TrimRight(resp.BlobReference.BlobURI, "/"), nil
}

func walkFiles(root string, pattern *regexp.Regexp) ([]string, error) {
	var out []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if pattern != nil && !pattern.MatchString(path) {
			return nil
		}
		out = append(out, path)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("datasets: walk %q: %w", root, err)
	}
	return out, nil
}

func uploadOneFile(ctx context.Context, containerSAS, localPath, blobName string) error {
	client, err := azblob.NewClientWithNoCredential(containerSAS, nil)
	if err != nil {
		return fmt.Errorf("datasets: blob client: %w", err)
	}
	containerName, err := containerNameFromSAS(containerSAS)
	if err != nil {
		return err
	}
	f, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("datasets: open %q: %w", localPath, err)
	}
	defer f.Close()
	if _, err := client.UploadFile(ctx, containerName, blobName, f, nil); err != nil {
		return fmt.Errorf("datasets: upload %q: %w", localPath, err)
	}
	return nil
}

// containerNameFromSAS extracts the container name from a container SAS URL
// of the form https://<account>.blob.core.windows.net/<container>?<sas>.
func containerNameFromSAS(sasURL string) (string, error) {
	q := strings.IndexByte(sasURL, '?')
	pathOnly := sasURL
	if q >= 0 {
		pathOnly = sasURL[:q]
	}
	scheme := strings.Index(pathOnly, "://")
	if scheme < 0 {
		return "", fmt.Errorf("datasets: bad SAS URL: %s", sasURL)
	}
	rest := pathOnly[scheme+3:]
	slash := strings.IndexByte(rest, '/')
	if slash < 0 || slash+1 >= len(rest) {
		return "", fmt.Errorf("datasets: SAS URL missing container: %s", sasURL)
	}
	return strings.TrimSuffix(rest[slash+1:], "/"), nil
}
