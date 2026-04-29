// Mirrors samples-dev/deployments/deploymentsBasics.ts.
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"

	"github.com/sambo/ai-projects-go/azaiprojects"
	"github.com/sambo/ai-projects-go/azaiprojects/deployments"
)

func main() {
	endpoint := os.Getenv("FOUNDRY_PROJECT_ENDPOINT")
	if endpoint == "" {
		log.Fatal("FOUNDRY_PROJECT_ENDPOINT is required")
	}
	modelPublisher := os.Getenv("MODEL_PUBLISHER")

	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		log.Fatalf("credential: %v", err)
	}
	project, err := azaiprojects.NewClient(endpoint, cred, nil)
	if err != nil {
		log.Fatalf("client: %v", err)
	}
	dep := project.Deployments()
	ctx := context.Background()

	// List all deployments.
	fmt.Println("List all deployments:")
	pager := dep.NewListPager(nil)
	var first deployments.ModelDeployment
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			log.Fatalf("list: %v", err)
		}
		for _, d := range page.Value {
			fmt.Printf("  name=%s modelPublisher=%s modelName=%s\n",
				d.Name, d.ModelPublisher, d.ModelName)
			if first.Name == "" {
				first = d
			}
		}
	}

	// Filter by model publisher (if env set).
	if modelPublisher != "" {
		fmt.Printf("Deployments by publisher %q:\n", modelPublisher)
		filtered := dep.NewListPager(&deployments.ListOptions{ModelPublisher: &modelPublisher})
		var n int
		for filtered.More() {
			page, err := filtered.NextPage(ctx)
			if err != nil {
				log.Fatalf("filtered list: %v", err)
			}
			n += len(page.Value)
		}
		fmt.Printf("  retrieved %d\n", n)
	}

	// Get a single deployment by name.
	if first.Name != "" {
		got, err := dep.Get(ctx, first.Name, nil)
		if err != nil {
			log.Fatalf("get: %v", err)
		}
		fmt.Printf("Got deployment: %s (model=%s)\n", got.Name, got.ModelName)
	}
}
