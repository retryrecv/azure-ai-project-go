# azure-ai-projects-go

A Go port of the [`@azure/ai-projects`](https://www.npmjs.com/package/@azure/ai-projects) JavaScript SDK for Azure AI Foundry. Hand-written REST client built on `github.com/Azure/azure-sdk-for-go/sdk/azcore`, mirroring the public TypeScript surface.

> Status: unofficial, local module. Not published to a public proxy.

## Install

```bash
go get github.com/retryrecv/azure-ai-projects-go
```

## Quick start

```go
package main

import (
    "context"
    "fmt"

    "github.com/Azure/azure-sdk-for-go/sdk/azidentity"
    "github.com/retryrecv/azure-ai-projects-go/azaiprojects"
)

func main() {
    cred, err := azidentity.NewDefaultAzureCredential(nil)
    if err != nil {
        panic(err)
    }

    client, err := azaiprojects.NewClient("https://<your-foundry-endpoint>", cred, nil)
    if err != nil {
        panic(err)
    }

    pager := client.Connections().NewListPager(nil)
    for pager.More() {
        page, err := pager.NextPage(context.Background())
        if err != nil {
            panic(err)
        }
        for _, c := range page.Value {
            fmt.Println(c.Name)
        }
    }
}
```

Set `FOUNDRY_PROJECT_ENDPOINT` and authenticate with the Azure CLI (`az login`) for the examples to pick up credentials via `DefaultAzureCredential`.

## Operation groups

Core (GA, `api-version=v1`):

- `connections` — list, get, getWithCredentials, getDefault
- `deployments` — list, get
- `datasets` — list, listVersions, get, createOrUpdate, delete, pendingUpload, getCredentials, uploadFile, uploadFolder
- `indexes` — list, listVersions, get, createOrUpdate, delete
- `agents` — list, get, create/update, version CRUD, manifest-import variants
- `evaluationRules` — list, get, createOrUpdate, delete

Beta (`Client.Beta()`, preview features):

- `skills`, `toolboxes`, `schedules`, `redTeams`, `memoryStores`, `insights`, `evaluators`, `evaluationTaxonomies`, `agents`

## Layout

```
azaiprojects/                  main package — AIProjectClient + sub-client accessors
  internal/shared/             pipeline helpers, paging, error mapping
  connections/                 ConnectionsClient
  deployments/                 DeploymentsClient
  datasets/                    DatasetsClient
  indexes/                     IndexesClient
  agents/                      AgentsClient
  evaluationrules/             EvaluationRulesClient
  beta/                        BetaOperations container
    skills/ toolboxes/ ...     beta sub-clients
examples/                      one runnable program per group
```

## Development

```bash
bash init.sh        # fetch deps, go build ./..., go test ./...
go build ./...
go test ./...
go vet ./...
go run ./examples/connections
```

Examples require `FOUNDRY_PROJECT_ENDPOINT` and Azure credentials at runtime; they `go build` without them.

## Reference

- TypeScript source: [`Azure/azure-sdk-for-js/sdk/ai/ai-projects`](https://github.com/Azure/azure-sdk-for-js/tree/main/sdk/ai/ai-projects)
- Default api-version: `v1`
- Default scope: `https://ai.azure.com/.default`

## License

MIT
