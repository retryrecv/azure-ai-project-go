// Mirrors samples-dev/indexes/indexesBasics.ts.
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/joho/godotenv"

	"github.com/sambo/ai-projects-go/azaiprojects"
	"github.com/sambo/ai-projects-go/azaiprojects/indexes"
)

func main() {
	_ = godotenv.Load()

	endpoint := os.Getenv("FOUNDRY_PROJECT_ENDPOINT")
	if endpoint == "" {
		log.Fatal("FOUNDRY_PROJECT_ENDPOINT is required")
	}
	indexName := envOr("AI_SEARCH_INDEX_NAME", "<index name>")
	indexVersion := envOr("AI_SEARCH_INDEX_VERSION", "<index version>")
	connectionName := envOr("AI_SEARCH_CONNECTION_NAME", "<connection name>")

	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		log.Fatalf("credential: %v", err)
	}
	project, err := azaiprojects.NewClient(endpoint, cred, nil)
	if err != nil {
		log.Fatalf("client: %v", err)
	}
	idx := project.Indexes()
	ctx := context.Background()

	const name = "my-azure-search-index"
	body := indexes.AzureAISearchIndex{
		Index: indexes.Index{
			Name:    name,
			Version: indexVersion,
			Type:    indexes.IndexTypeAzureSearch,
		},
		IndexName:      indexName,
		ConnectionName: connectionName,
	}

	created, err := idx.CreateOrUpdate(ctx, name, "1.0", body, nil)
	if err != nil {
		log.Fatalf("createOrUpdate: %v", err)
	}
	fmt.Printf("Created index: name=%s version=%s type=%s\n", created.Name, created.Version, created.Type)

	got, err := idx.Get(ctx, name, created.Version, nil)
	if err != nil {
		log.Fatalf("get: %v", err)
	}
	fmt.Printf("Get index: name=%s version=%s\n", got.Name, got.Version)

	fmt.Println("List versions:")
	versions := idx.NewListVersionsPager(name, nil)
	for versions.More() {
		page, err := versions.NextPage(ctx)
		if err != nil {
			log.Fatalf("list versions: %v", err)
		}
		for _, v := range page.Value {
			fmt.Printf("  %s/%s\n", v.Name, v.Version)
		}
	}

	fmt.Println("List all:")
	all := idx.NewListPager(nil)
	for all.More() {
		page, err := all.NextPage(ctx)
		if err != nil {
			log.Fatalf("list: %v", err)
		}
		for _, v := range page.Value {
			fmt.Printf("  %s/%s (%s)\n", v.Name, v.Version, v.Type)
		}
	}

	if _, err := idx.Delete(ctx, name, created.Version, nil); err != nil {
		log.Fatalf("delete: %v", err)
	}
	fmt.Println("Index operations completed successfully")
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
