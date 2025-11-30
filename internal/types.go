package internal

import "time"

// CatalogDetail represents detailed information about a software in the catalog
type CatalogDetail struct {
	Name        string                  `json:"name"`
	Description string                  `json:"description"`
	Website     string                  `json:"website"`
	Icon        string                  `json:"icon"`
	Category    string                  `json:"category"`
	Tags        []string                `json:"tags"`
	Inputs      map[string]InputConfig  `json:"inputs"`
	Services    map[string]ServiceInfo  `json:"services"`
}

// InputConfig represents a configurable input for software deployment
type InputConfig struct {
	Label       string `json:"label"`
	Placeholder string `json:"placeholder"`
	Required    bool   `json:"required"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Default     string `json:"default"`
}

// ServiceInfo represents information about a service component
type ServiceInfo struct {
	Exposed       bool   `json:"exposed"`
	Stateless     bool   `json:"stateless"`
	Internal      bool   `json:"internal"`
	ManagedOption string `json:"managed_option"`
}

// LocalDeployment represents a local Docker deployment
type LocalDeployment struct {
	ID        string            `json:"id"`
	Software  string            `json:"software"`
	Target    string            `json:"target"`
	Status    string            `json:"status"`
	Directory string            `json:"directory"`
	Port      int               `json:"port"`
	Inputs    map[string]string `json:"inputs"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
}

// DeploymentsFile represents the structure of the deployments.json file
type DeploymentsFile struct {
	Deployments []LocalDeployment `json:"deployments"`
}
