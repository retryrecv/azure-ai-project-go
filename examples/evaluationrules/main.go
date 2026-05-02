// Demonstrates the EvaluationRules operation group.
//
// The TS sample (samples-dev/evaluations/continuousEvaluationRule.ts) creates
// an eval via the OpenAI passthrough client (project.getOpenAIClient), which
// is out of scope for this Go port. This example takes a pre-existing eval ID
// via FOUNDRY_EVAL_ID and walks CreateOrUpdate -> Get -> List -> Delete.
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/joho/godotenv"

	"github.com/retryrecv/azure-ai-projects-go/azaiprojects"
	"github.com/retryrecv/azure-ai-projects-go/azaiprojects/evaluationrules"
)

func main() {
	_ = godotenv.Load()

	endpoint := os.Getenv("FOUNDRY_PROJECT_ENDPOINT")
	if endpoint == "" {
		log.Fatal("FOUNDRY_PROJECT_ENDPOINT is required")
	}
	evalID := envOr("FOUNDRY_EVAL_ID", "<eval id>")
	agentName := envOr("FOUNDRY_AGENT_NAME", "my-eval-rule-agent")

	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		log.Fatalf("credential: %v", err)
	}
	project, err := azaiprojects.NewClient(endpoint, cred, nil)
	if err != nil {
		log.Fatalf("client: %v", err)
	}
	er := project.EvaluationRules()
	ctx := context.Background()

	const ruleID = "my-continuous-eval-rule"
	maxHourly := int32(100)
	rule := evaluationrules.EvaluationRule{
		DisplayName: "My Continuous Eval Rule",
		Description: "An eval rule that runs on agent response completions",
		EventType:   evaluationrules.EvaluationRuleEventTypeResponseCompleted,
		Enabled:     true,
		Filter:      &evaluationrules.EvaluationRuleFilter{AgentName: agentName},
		Action: evaluationrules.EvaluationRuleActionValue{
			Value: evaluationrules.ContinuousEvaluationRuleAction{
				Type:          evaluationrules.EvaluationRuleActionTypeContinuousEvaluation,
				EvalID:        evalID,
				MaxHourlyRuns: &maxHourly,
			},
		},
	}

	created, err := er.CreateOrUpdate(ctx, ruleID, rule, nil)
	if err != nil {
		log.Fatalf("createOrUpdate: %v", err)
	}
	fmt.Printf("Created evaluation rule id=%s displayName=%s eventType=%s\n",
		created.ID, created.DisplayName, created.EventType)

	got, err := er.Get(ctx, created.ID, nil)
	if err != nil {
		log.Fatalf("get: %v", err)
	}
	fmt.Printf("Got evaluation rule id=%s enabled=%t\n", got.ID, got.Enabled)

	fmt.Println("List evaluation rules:")
	pager := er.NewListPager(nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			log.Fatalf("list: %v", err)
		}
		for _, r := range page.Value {
			fmt.Printf("  id=%s displayName=%s actionType=%s\n", r.ID, r.DisplayName, r.Action.Type)
		}
	}

	if _, err := er.Delete(ctx, created.ID, nil); err != nil {
		log.Fatalf("delete: %v", err)
	}
	fmt.Println("Evaluation rule deleted")
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
