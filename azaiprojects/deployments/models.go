package deployments

// DeploymentType is the discriminator for Deployment.
type DeploymentType string

const (
	// DeploymentTypeModel is currently the only known deployment type.
	DeploymentTypeModel DeploymentType = "ModelDeployment"
)

// Deployment is the base view of a deployment. ModelDeployment carries the
// fields specific to model deployments.
type Deployment struct {
	Type DeploymentType `json:"type"`
	Name string         `json:"name"`
}

// ModelDeploymentSku captures Sku information returned with a ModelDeployment.
type ModelDeploymentSku struct {
	Capacity int32  `json:"capacity"`
	Family   string `json:"family"`
	Name     string `json:"name"`
	Size     string `json:"size"`
	Tier     string `json:"tier"`
}

// ModelDeployment is a model-flavored deployment.
type ModelDeployment struct {
	Deployment
	ModelName      string             `json:"modelName"`
	ModelVersion   string             `json:"modelVersion"`
	ModelPublisher string             `json:"modelPublisher"`
	Capabilities   map[string]string  `json:"capabilities,omitempty"`
	Sku            ModelDeploymentSku `json:"sku"`
	// ConnectionName is the connection the deployment comes from. Optional.
	ConnectionName string `json:"connectionName,omitempty"`
}
