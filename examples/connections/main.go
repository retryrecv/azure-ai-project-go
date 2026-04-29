// Mirrors samples-dev/connections/connectionsBasics.ts.
//
// Required env: FOUNDRY_PROJECT_ENDPOINT and Azure credentials available
// to azidentity.NewDefaultAzureCredential.
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"

	"github.com/sambo/ai-projects-go/azaiprojects"
	"github.com/sambo/ai-projects-go/azaiprojects/connections"
)

func main() {
	endpoint := os.Getenv("FOUNDRY_PROJECT_ENDPOINT")
	if endpoint == "" {
		log.Fatal("FOUNDRY_PROJECT_ENDPOINT is required")
	}
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		log.Fatalf("credential: %v", err)
	}
	project, err := azaiprojects.NewClient(endpoint, cred, nil)
	if err != nil {
		log.Fatalf("client: %v", err)
	}
	conns := project.Connections()
	ctx := context.Background()

	// List all connections.
	pager := conns.NewListPager(nil)
	var first connections.Connection
	var names []string
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			log.Fatalf("list page: %v", err)
		}
		for _, c := range page.Value {
			names = append(names, c.Name)
			if first.Name == "" {
				first = c
			}
		}
	}
	fmt.Printf("Retrieved connections: %v\n", names)

	if first.Name == "" {
		fmt.Println("No connections found; nothing else to do.")
		return
	}

	// Get the first connection (no credentials).
	got, err := conns.Get(ctx, first.Name, nil)
	if err != nil {
		log.Fatalf("get: %v", err)
	}
	fmt.Printf("connection.type: %s connection.name: %s connection.target: %s\n",
		got.Type, got.Name, got.Target)

	// Get the same connection with credentials.
	withCreds, err := conns.GetWithCredentials(ctx, first.Name, nil)
	if err != nil {
		log.Fatalf("get with credentials: %v", err)
	}
	fmt.Printf("credentials.type: %s\n", withCreds.Credentials.Type)

	// List only AzureOpenAI default connections.
	defaultPager := conns.NewListPager(&connections.ListOptions{
		ConnectionType:    ptr(connections.ConnectionTypeAzureOpenAI),
		DefaultConnection: ptr(true),
	})
	var aoaiCount int
	for defaultPager.More() {
		page, err := defaultPager.NextPage(ctx)
		if err != nil {
			log.Fatalf("default list page: %v", err)
		}
		aoaiCount += len(page.Value)
	}
	fmt.Printf("Retrieved %d Azure OpenAI default connections\n", aoaiCount)

	// Get default AzureOpenAI connection with credentials.
	def, err := conns.GetDefault(ctx, connections.ConnectionTypeAzureOpenAI,
		&connections.GetDefaultOptions{IncludeCredentials: true})
	if err != nil {
		log.Printf("get default: %v", err)
		return
	}
	fmt.Printf("Retrieved default connection: name=%s type=%s\n", def.Name, def.Type)
}

func ptr[T any](v T) *T { return &v }
