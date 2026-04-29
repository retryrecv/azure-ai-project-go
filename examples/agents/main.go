// Mirrors samples-dev/agents/agentCurd.ts.
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/joho/godotenv"

	"github.com/sambo/ai-projects-go/azaiprojects"
	"github.com/sambo/ai-projects-go/azaiprojects/agents"
)

func main() {
	_ = godotenv.Load()

	endpoint := os.Getenv("FOUNDRY_PROJECT_ENDPOINT")
	if endpoint == "" {
		log.Fatal("FOUNDRY_PROJECT_ENDPOINT is required")
	}
	deploymentName := envOr("FOUNDRY_MODEL_NAME", "<model deployment name>")

	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		log.Fatalf("credential: %v", err)
	}
	project, err := azaiprojects.NewClient(endpoint, cred, nil)
	if err != nil {
		log.Fatalf("client: %v", err)
	}
	ag := project.Agents()
	ctx := context.Background()

	const name1 = "go-crud-agent1"
	const name2 = "go-crud-agent2"

	v1, err := ag.CreateVersion(ctx, name1,
		agents.PromptAgentDefinition{Kind: agents.AgentKindPrompt, Model: deploymentName}, nil)
	if err != nil {
		log.Fatalf("createVersion %s: %v", name1, err)
	}
	fmt.Printf("Created agent id=%s version=%s name=%s\n", v1.ID, v1.Version, v1.Name)

	v2, err := ag.CreateVersion(ctx, name2,
		agents.PromptAgentDefinition{Kind: agents.AgentKindPrompt, Model: deploymentName}, nil)
	if err != nil {
		log.Fatalf("createVersion %s: %v", name2, err)
	}
	fmt.Printf("Created agent id=%s version=%s name=%s\n", v2.ID, v2.Version, v2.Name)

	got, err := ag.GetVersion(ctx, v1.Name, v1.Version, nil)
	if err != nil {
		log.Fatalf("getVersion: %v", err)
	}
	fmt.Printf("Retrieved agent id=%s version=%s name=%s\n", got.ID, got.Version, got.Name)

	latest, err := ag.Get(ctx, v1.Name, nil)
	if err != nil {
		log.Fatalf("get: %v", err)
	}
	fmt.Printf("Retrieved latest agent id=%s name=%s\n", latest.ID, latest.Versions.Latest.Name)

	fmt.Println("List versions:")
	pager := ag.NewListVersionsPager(v1.Name, nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			log.Fatalf("listVersions: %v", err)
		}
		for _, av := range page.Data {
			fmt.Printf("  agent id=%s name=%s version=%s\n", av.ID, av.Name, av.Version)
		}
	}

	r1, err := ag.DeleteVersion(ctx, v1.Name, v1.Version, nil)
	if err != nil {
		log.Fatalf("deleteVersion %s: %v", v1.Name, err)
	}
	fmt.Printf("Deleted agent name=%s version=%s result=%t\n", r1.Name, v1.Version, r1.Deleted)

	r2, err := ag.DeleteVersion(ctx, v2.Name, v2.Version, nil)
	if err != nil {
		log.Fatalf("deleteVersion %s: %v", v2.Name, err)
	}
	fmt.Printf("Deleted agent name=%s version=%s result=%t\n", r2.Name, v2.Version, r2.Deleted)
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
