// Mirrors samples-dev/datasets/datasetsBasics.ts (excluding uploadFile/uploadFolder).
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/joho/godotenv"

	"github.com/sambo/ai-projects-go/azaiprojects"
	"github.com/sambo/ai-projects-go/azaiprojects/datasets"
)

func main() {
	_ = godotenv.Load()

	endpoint := os.Getenv("FOUNDRY_PROJECT_ENDPOINT")
	if endpoint == "" {
		log.Fatal("FOUNDRY_PROJECT_ENDPOINT is required")
	}
	connectionName := envOr("AZURE_STORAGE_CONNECTION_NAME", "<storage connection name>")

	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		log.Fatalf("credential: %v", err)
	}
	project, err := azaiprojects.NewClient(endpoint, cred, nil)
	if err != nil {
		log.Fatalf("client: %v", err)
	}
	ds := project.Datasets()
	ctx := context.Background()

	const name = "sample-dataset-folder"
	const version = "1.0"
	const folderPath = "examples/datasets/sample_folder"

	created, err := ds.UploadFolder(ctx, name, version, folderPath, &datasets.UploadOptions{
		ConnectionName: connectionName,
	})
	if err != nil {
		log.Fatalf("uploadFolder: %v", err)
	}
	fmt.Printf("Uploaded folder; created dataset: name=%s version=%s type=%s\n",
		created.Name, created.Version, created.Type)

	cred2, err := ds.GetCredentials(ctx, name, version, nil)
	if err != nil {
		log.Fatalf("getCredentials: %v", err)
	}
	fmt.Printf("Dataset SAS: %s\n", cred2.BlobReference.Credential.SASURI)

	got, err := ds.Get(ctx, name, version, nil)
	if err != nil {
		log.Fatalf("get: %v", err)
	}
	fmt.Printf("Got dataset: %s/%s (%s)\n", got.Name, got.Version, got.Type)

	versions := ds.NewListVersionsPager(name, nil)
	for versions.More() {
		page, err := versions.NextPage(ctx)
		if err != nil {
			log.Fatalf("listVersions: %v", err)
		}
		for _, v := range page.Value {
			fmt.Printf("  version: %s/%s\n", v.Name, v.Version)
		}
	}

	all := ds.NewListPager(nil)
	for all.More() {
		page, err := all.NextPage(ctx)
		if err != nil {
			log.Fatalf("list: %v", err)
		}
		for _, v := range page.Value {
			fmt.Printf("  dataset: %s/%s\n", v.Name, v.Version)
		}
	}

	if _, err := ds.Delete(ctx, name, version, nil); err != nil {
		log.Fatalf("delete: %v", err)
	}
	fmt.Println("Dataset deleted")
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
