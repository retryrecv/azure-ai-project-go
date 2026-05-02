# azure-ai-projects-go

Go port of the `@azure/ai-projects` JavaScript SDK. Hand-written REST-style client built on `github.com/Azure/azure-sdk-for-go/sdk/azcore`. Mirrors the public TypeScript surface (`AIProjectClient` + operation groups: connections, deployments, datasets, indexes for the core scope).

## Reference

- TS source: `/Users/goubo/projects/azure-sdk-for-js/sdk/ai/ai-projects`
- Public API: `src/index.ts` exports
- Usage examples: `samples-dev/<group>/<name>Basics.ts`
- REST URL templates: `src/api/<group>/operations.ts`
- Default api-version: `v1`. Default scope: `https://ai.azure.com/.default`.

## Module

```
github.com/retryrecv/azure-ai-projects-go
```

(Local module — not published. Importable from `examples/` and tests.)

## Layout

```
azaiprojects/                  # main package
  client.go                    # AIProjectClient (constructor + sub-client accessors)
  options.go                   # ClientOptions, api-version constants
  internal/
    shared/                    # pipeline helpers, paging, error mapping
  connections/                 # ConnectionsClient — list, get, getWithCredentials, getDefault
  deployments/                 # DeploymentsClient — list, get
  datasets/                    # DatasetsClient — list, listVersions, get, createOrUpdate, delete, pendingUpload, getCredentials, uploadFile, uploadFolder
  indexes/                     # IndexesClient — list, listVersions, get, createOrUpdate, delete
examples/                      # one runnable program per group, mirrors samples-dev/*Basics.ts
```

## Startup sequence (every session)

1. `pwd` — confirm in `/Users/goubo/projects/azure-ai-projects-go`
2. `cat claude-progress.txt && git log --oneline -10`
3. `bash init.sh` — fetches deps, runs `go build ./...` and `go test ./...` smoke test
4. If smoke test fails, fix before anything else
5. Read `feature_list.json`, pick lowest-priority task with `"passes": false`
6. Execute every step in `steps`; only flip `passes: true` after end-to-end confirmation
7. After the task: `git add -A && git commit -m "feat: <task-id> — <summary>"`, append to `claude-progress.txt`, continue immediately to next task
8. Stop only when all tasks have `passes: true`

## End-of-session

```
git add -A && git commit -m "feat: <task-id> — <summary>"
# append to claude-progress.txt:
# --- Session <N> ---
# Feature: <id>
# What I did: <1-2 sentences>
# Next: <id of next failing feature>
# Issues: <or "none">
```

## Commands

- Build: `go build ./...`
- Test: `go test ./...`
- Vet: `go vet ./...`
- Run an example: `go run ./examples/connections` (requires env: `FOUNDRY_PROJECT_ENDPOINT` + Azure credentials)

## Test approach

- **Unit tests** with mocked `policy.Transporter` for every operation — assert URL path, query params (`api-version`), method, and request/response body shape.
- **Compile checks** for examples — every `examples/<group>/main.go` must `go build`. They are not run against a live service in CI.
- No live integration tests in the harness loop. A future task can add recorded HTTP fixtures.

## Out of scope (for now)

`agents`, `beta.*`, `evaluationRules`, `telemetry`, OpenAI client passthrough (`getOpenAIClient`). These are tracked as future tasks but not in the initial feature list.
