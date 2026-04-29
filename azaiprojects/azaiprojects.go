// Package azaiprojects is a Go port of @azure/ai-projects.
//
// This bootstrap file exists so `go mod tidy` retains azcore as a direct
// dependency before the rest of the package is fleshed out (task
// package-skeleton-builds).
package azaiprojects

import _ "github.com/Azure/azure-sdk-for-go/sdk/azcore"
