package azaiprojects

// Held until the runtime pipeline lands in client.go (task aiproject-client-constructor)
// so `go mod tidy` keeps azcore as a direct require.
import _ "github.com/Azure/azure-sdk-for-go/sdk/azcore"

const (
	ModuleName    = "azaiprojects"
	ModuleVersion = "0.1.0"
)
