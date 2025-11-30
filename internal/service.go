package internal

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gofr.dev/pkg/gofr"
)

const (
	catalogRepoURL = "https://github.com/opengittr/opensourcer-catalog.git"
)

// Service handles CLI operations
type Service struct {
	configPath  string
	catalogPath string
	deployments []LocalDeployment
}

// NewService creates a new CLI service
func NewService() *Service {
	homeDir, _ := os.UserHomeDir()
	configPath := filepath.Join(homeDir, ".opensourcer")
	catalogPath := filepath.Join(configPath, "catalog")

	// Ensure config directory exists
	_ = os.MkdirAll(configPath, 0755)

	s := &Service{
		configPath:  configPath,
		catalogPath: catalogPath,
	}

	// Load existing deployments
	s.loadDeployments()

	return s
}

// getArg extracts the first positional argument after the subcommand
func getArg(c *gofr.Context) string {
	if len(os.Args) >= 3 {
		arg := os.Args[2]
		if !strings.HasPrefix(arg, "-") {
			return arg
		}
	}
	return ""
}

// ListCatalog lists available software in the catalog
func (s *Service) ListCatalog(c *gofr.Context) (interface{}, error) {
	if _, err := os.Stat(s.catalogPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("catalog not found. Run 'opensourcer update' first")
	}

	entries, err := os.ReadDir(s.catalogPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read catalog: %w", err)
	}

	var output strings.Builder
	output.WriteString("\nAvailable Software\n")
	output.WriteString(strings.Repeat("-", 60) + "\n\n")

	count := 0
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), "_") {
			continue
		}

		slug := entry.Name()
		detail, err := s.getCatalogDetail(slug)
		if err != nil {
			continue
		}

		output.WriteString(fmt.Sprintf("  %-15s %s\n", slug, detail.Name))
		output.WriteString(fmt.Sprintf("                  %s\n", detail.Description))
		output.WriteString(fmt.Sprintf("                  Category: %s\n\n", detail.Category))
		count++
	}

	output.WriteString(fmt.Sprintf("Total: %d software available\n\n", count))
	output.WriteString("Use 'opensourcer info <software>' for details\n")
	output.WriteString("Use 'opensourcer deploy <software>' to deploy\n")

	return output.String(), nil
}

// GetInfo shows detailed information about a software
func (s *Service) GetInfo(c *gofr.Context) (interface{}, error) {
	software := getArg(c)
	if software == "" {
		return nil, fmt.Errorf("usage: opensourcer info <software>")
	}

	detail, err := s.getCatalogDetail(software)
	if err != nil {
		return nil, err
	}

	var output strings.Builder
	output.WriteString(fmt.Sprintf("\n%s\n", detail.Name))
	output.WriteString(strings.Repeat("-", 60) + "\n\n")
	output.WriteString(fmt.Sprintf("  %s\n\n", detail.Description))
	output.WriteString(fmt.Sprintf("  Website: %s\n", detail.Website))
	output.WriteString(fmt.Sprintf("  Category: %s\n", detail.Category))
	output.WriteString(fmt.Sprintf("  Tags: %s\n", strings.Join(detail.Tags, ", ")))

	if len(detail.Inputs) > 0 {
		output.WriteString("\n  Configuration inputs:\n")
		for key, input := range detail.Inputs {
			req := ""
			if input.Required {
				req = " (required)"
			}
			output.WriteString(fmt.Sprintf("    --%s: %s%s\n", key, input.Label, req))
		}
	}

	output.WriteString(fmt.Sprintf("\nDeploy locally: opensourcer deploy %s\n", software))

	return output.String(), nil
}

// Deploy deploys software locally or to cloud
func (s *Service) Deploy(c *gofr.Context) (interface{}, error) {
	software := getArg(c)
	if software == "" {
		return nil, fmt.Errorf("usage: opensourcer deploy <software> [--target local|aws]")
	}

	detail, err := s.getCatalogDetail(software)
	if err != nil {
		return nil, err
	}

	// Parse inputs from flags
	inputs := make(map[string]string)
	for key := range detail.Inputs {
		if val := c.Param(key); val != "" {
			inputs[key] = val
		}
	}

	target := c.Param("target")
	if target == "" {
		target = "local"
	}

	switch target {
	case "local":
		return s.deployLocal(c, software, detail, inputs)
	case "aws":
		return nil, fmt.Errorf("AWS deployment not yet implemented. Use --target local")
	default:
		return nil, fmt.Errorf("unknown target: %s", target)
	}
}

// List shows all deployments
func (s *Service) List(c *gofr.Context) (interface{}, error) {
	if len(s.deployments) == 0 {
		return "\nNo deployments found.\n\nUse 'opensourcer deploy <software>' to create one.\n", nil
	}

	var output strings.Builder
	output.WriteString("\nYour Deployments\n")
	output.WriteString(strings.Repeat("-", 70) + "\n\n")

	for _, d := range s.deployments {
		statusIcon := "[running]"
		if d.Status == "stopped" {
			statusIcon = "[stopped]"
		}

		output.WriteString(fmt.Sprintf("  %s %-12s  %-10s %s\n", statusIcon, d.Software, d.Target, d.Status))
		output.WriteString(fmt.Sprintf("     ID: %s\n", d.ID[:8]))
		if d.Port > 0 {
			output.WriteString(fmt.Sprintf("     URL: http://localhost:%d\n", d.Port))
		}
		output.WriteString(fmt.Sprintf("     Created: %s\n\n", d.CreatedAt.Format("2006-01-02 15:04")))
	}

	return output.String(), nil
}

// Logs shows logs for a deployment
func (s *Service) Logs(c *gofr.Context) (interface{}, error) {
	software := getArg(c)
	if software == "" {
		return nil, fmt.Errorf("usage: opensourcer logs <software>")
	}

	deployment := s.findDeployment(software)
	if deployment == nil {
		return nil, fmt.Errorf("deployment '%s' not found", software)
	}

	return s.getDockerLogs(deployment)
}

// Stop stops a running deployment
func (s *Service) Stop(c *gofr.Context) (interface{}, error) {
	software := getArg(c)
	if software == "" {
		return nil, fmt.Errorf("usage: opensourcer stop <software>")
	}

	deployment := s.findDeployment(software)
	if deployment == nil {
		return nil, fmt.Errorf("deployment '%s' not found", software)
	}

	return s.stopDocker(deployment)
}

// Start starts a stopped deployment
func (s *Service) Start(c *gofr.Context) (interface{}, error) {
	software := getArg(c)
	if software == "" {
		return nil, fmt.Errorf("usage: opensourcer start <software>")
	}

	deployment := s.findDeployment(software)
	if deployment == nil {
		return nil, fmt.Errorf("deployment '%s' not found", software)
	}

	return s.startDocker(deployment)
}

// Destroy removes a deployment completely
func (s *Service) Destroy(c *gofr.Context) (interface{}, error) {
	software := getArg(c)
	if software == "" {
		return nil, fmt.Errorf("usage: opensourcer destroy <software>")
	}

	deployment := s.findDeployment(software)
	if deployment == nil {
		return nil, fmt.Errorf("deployment '%s' not found", software)
	}

	if _, err := s.destroyDocker(deployment); err != nil {
		return nil, err
	}

	s.removeDeployment(deployment.ID)

	return fmt.Sprintf("\nDestroyed '%s' deployment\n", software), nil
}

// Helper functions

func (s *Service) getCatalogDetail(software string) (*CatalogDetail, error) {
	appJSONPath := filepath.Join(s.catalogPath, software, "app.json")
	data, err := os.ReadFile(appJSONPath)
	if err != nil {
		return nil, fmt.Errorf("software '%s' not found in catalog", software)
	}

	var detail CatalogDetail
	if err := json.Unmarshal(data, &detail); err != nil {
		return nil, fmt.Errorf("invalid app.json for '%s'", software)
	}

	return &detail, nil
}

func (s *Service) getComposePath(software string) string {
	return filepath.Join(s.catalogPath, software, "docker-compose.yaml")
}

func (s *Service) loadDeployments() {
	deployFile := filepath.Join(s.configPath, "deployments.json")
	data, err := os.ReadFile(deployFile)
	if err != nil {
		return
	}

	var file DeploymentsFile
	if err := json.Unmarshal(data, &file); err != nil {
		return
	}

	s.deployments = file.Deployments
}

func (s *Service) saveDeployments() {
	file := DeploymentsFile{Deployments: s.deployments}
	data, _ := json.MarshalIndent(file, "", "  ")
	deployFile := filepath.Join(s.configPath, "deployments.json")
	_ = os.WriteFile(deployFile, data, 0644)
}

func (s *Service) addDeployment(d LocalDeployment) {
	s.deployments = append(s.deployments, d)
	s.saveDeployments()
}

func (s *Service) findDeployment(software string) *LocalDeployment {
	for i := range s.deployments {
		if s.deployments[i].Software == software {
			return &s.deployments[i]
		}
	}
	return nil
}

func (s *Service) updateDeploymentStatus(id, status string) {
	for i := range s.deployments {
		if s.deployments[i].ID == id {
			s.deployments[i].Status = status
			s.deployments[i].UpdatedAt = time.Now()
			s.saveDeployments()
			return
		}
	}
}

func (s *Service) removeDeployment(id string) {
	for i := range s.deployments {
		if s.deployments[i].ID == id {
			s.deployments = append(s.deployments[:i], s.deployments[i+1:]...)
			s.saveDeployments()
			return
		}
	}
}

func generatePassword(length int) string {
	bytes := make([]byte, length/2+1)
	rand.Read(bytes)
	return strings.ToUpper(hex.EncodeToString(bytes)[:length])
}
