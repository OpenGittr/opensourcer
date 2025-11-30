package internal

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"gofr.dev/pkg/gofr"
)

func (s *Service) deployLocal(c *gofr.Context, software string, detail *CatalogDetail, inputs map[string]string) (interface{}, error) {
	// Check if Docker is available
	if err := checkDockerAvailable(); err != nil {
		return nil, fmt.Errorf("docker is required for local deployment: %w", err)
	}

	// Create deployment directory
	deployDir := filepath.Join(s.configPath, "deployments", software)
	if err := os.MkdirAll(deployDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create deployment directory: %w", err)
	}

	// Copy all files from catalog directory to deployment directory
	catalogDir := filepath.Join(s.catalogPath, software)
	if err := copyDir(catalogDir, deployDir); err != nil {
		return nil, fmt.Errorf("failed to copy catalog files: %w", err)
	}

	// Read docker-compose.yaml content for port detection
	composePath := filepath.Join(deployDir, "docker-compose.yaml")
	composeContent, err := os.ReadFile(composePath)
	if err != nil {
		return nil, fmt.Errorf("docker-compose.yaml not found for '%s'", software)
	}

	// Prepare environment variables
	envVars := prepareEnvVars(detail, inputs)

	// Write .env file
	envContent := buildEnvFile(envVars)
	envPath := filepath.Join(deployDir, ".env")
	if err := os.WriteFile(envPath, []byte(envContent), 0644); err != nil {
		return nil, fmt.Errorf("failed to write .env file: %w", err)
	}

	// Find exposed port from compose file
	port := findExposedPort(string(composeContent))

	// Run docker-compose up
	var output strings.Builder
	output.WriteString(fmt.Sprintf("\n🚀 Deploying %s locally...\n\n", detail.Name))

	cmd := exec.Command("docker", "compose", "-f", composePath, "up", "-d")
	cmd.Dir = deployDir
	cmd.Env = append(os.Environ(), envVarsToSlice(envVars)...)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("docker compose failed: %s", stderr.String())
	}

	// Create deployment record
	deployment := LocalDeployment{
		ID:        uuid.New().String(),
		Software:  software,
		Target:    "local",
		Status:    "running",
		Directory: deployDir,
		Port:      port,
		Inputs:    inputs,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	s.addDeployment(deployment)

	output.WriteString("✅ Deployment successful!\n\n")
	output.WriteString(fmt.Sprintf("  Software: %s\n", detail.Name))
	output.WriteString(fmt.Sprintf("  Status: running\n"))
	if port > 0 {
		output.WriteString(fmt.Sprintf("  URL: http://localhost:%d\n", port))
	}
	output.WriteString(fmt.Sprintf("  Directory: %s\n", deployDir))

	// Show generated credentials if any
	if pwd, ok := envVars["DB_PASSWORD"]; ok {
		output.WriteString(fmt.Sprintf("\n  Generated DB Password: %s\n", pwd))
	}
	if pwd, ok := envVars["ADMIN_PASSWORD"]; ok {
		output.WriteString(fmt.Sprintf("  Generated Admin Password: %s\n", pwd))
	}
	if pwd, ok := envVars["BASIC_AUTH_PASSWORD"]; ok && inputs["basic_auth_password"] == "" {
		output.WriteString(fmt.Sprintf("  Generated Auth Password: %s\n", pwd))
	}

	output.WriteString("\nUseful commands:\n")
	output.WriteString(fmt.Sprintf("  opensourcer logs %s    - View logs\n", software))
	output.WriteString(fmt.Sprintf("  opensourcer stop %s    - Stop deployment\n", software))
	output.WriteString(fmt.Sprintf("  opensourcer destroy %s - Remove deployment\n", software))

	return output.String(), nil
}

func (s *Service) getDockerLogs(deployment *LocalDeployment) (interface{}, error) {
	composePath := filepath.Join(deployment.Directory, "docker-compose.yaml")

	cmd := exec.Command("docker", "compose", "-f", composePath, "logs", "--tail", "100")
	cmd.Dir = deployment.Directory

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to get logs: %w", err)
	}

	return fmt.Sprintf("\n📋 Logs for %s\n%s\n%s", deployment.Software, strings.Repeat("─", 60), string(output)), nil
}

func (s *Service) stopDocker(deployment *LocalDeployment) (interface{}, error) {
	composePath := filepath.Join(deployment.Directory, "docker-compose.yaml")

	cmd := exec.Command("docker", "compose", "-f", composePath, "stop")
	cmd.Dir = deployment.Directory

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to stop containers: %w", err)
	}

	s.updateDeploymentStatus(deployment.ID, "stopped")

	return fmt.Sprintf("\n⏹️  Stopped '%s'\n\nUse 'opensourcer start %s' to restart\n", deployment.Software, deployment.Software), nil
}

func (s *Service) startDocker(deployment *LocalDeployment) (interface{}, error) {
	composePath := filepath.Join(deployment.Directory, "docker-compose.yaml")

	cmd := exec.Command("docker", "compose", "-f", composePath, "start")
	cmd.Dir = deployment.Directory

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to start containers: %w", err)
	}

	s.updateDeploymentStatus(deployment.ID, "running")

	var output strings.Builder
	output.WriteString(fmt.Sprintf("\n✅ Started '%s'\n", deployment.Software))
	if deployment.Port > 0 {
		output.WriteString(fmt.Sprintf("\n  URL: http://localhost:%d\n", deployment.Port))
	}

	return output.String(), nil
}

func (s *Service) destroyDocker(deployment *LocalDeployment) (interface{}, error) {
	composePath := filepath.Join(deployment.Directory, "docker-compose.yaml")

	// Stop and remove containers, networks, volumes
	cmd := exec.Command("docker", "compose", "-f", composePath, "down", "-v")
	cmd.Dir = deployment.Directory

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to destroy containers: %w", err)
	}

	// Remove deployment directory
	_ = os.RemoveAll(deployment.Directory)

	return nil, nil
}

func checkDockerAvailable() error {
	cmd := exec.Command("docker", "info")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker is not running or not installed")
	}
	return nil
}

func prepareEnvVars(detail *CatalogDetail, inputs map[string]string) map[string]string {
	envVars := make(map[string]string)

	// Set defaults for common variables
	envVars["DB_PASSWORD"] = generatePassword(16)
	envVars["ADMIN_PASSWORD"] = generatePassword(12)
	envVars["SECRET_KEY"] = generatePassword(48)

	// Map input keys to environment variable names
	keyMapping := map[string]string{
		"domain":              "DOMAIN",
		"timezone":            "TIMEZONE",
		"basic_auth_user":     "BASIC_AUTH_USER",
		"basic_auth_password": "BASIC_AUTH_PASSWORD",
		"admin_user":          "ADMIN_USER",
		"admin_password":      "ADMIN_PASSWORD",
		"admin_email":         "ADMIN_EMAIL",
		"site_title":          "SITE_TITLE",
	}

	// Apply user inputs
	for key, value := range inputs {
		if value == "" {
			continue
		}

		envKey := strings.ToUpper(strings.ReplaceAll(key, "-", "_"))
		if mapped, ok := keyMapping[key]; ok {
			envKey = mapped
		}
		envVars[envKey] = value
	}

	// Generate passwords for required fields if not provided
	if detail.Inputs != nil {
		for key, input := range detail.Inputs {
			envKey := strings.ToUpper(strings.ReplaceAll(key, "-", "_"))
			if mapped, ok := keyMapping[key]; ok {
				envKey = mapped
			}

			if _, exists := envVars[envKey]; !exists && input.Type == "password" {
				envVars[envKey] = generatePassword(12)
			}
		}
	}

	// Set localhost domain if not provided
	if _, ok := envVars["DOMAIN"]; !ok {
		envVars["DOMAIN"] = "localhost"
	}

	return envVars
}

func buildEnvFile(envVars map[string]string) string {
	var lines []string
	for key, value := range envVars {
		lines = append(lines, fmt.Sprintf("%s=%s", key, value))
	}
	return strings.Join(lines, "\n")
}

func envVarsToSlice(envVars map[string]string) []string {
	var result []string
	for key, value := range envVars {
		result = append(result, fmt.Sprintf("%s=%s", key, value))
	}
	return result
}

func findExposedPort(composeContent string) int {
	// Common default ports - order matters, check more specific first
	portMap := map[string]int{
		"2368": 2368, // ghost
		"3000": 3000, // gitea
		"3001": 3001, // uptime-kuma
		"5678": 5678, // n8n
		"8000": 8000, // plausible
		"8065": 8065, // mattermost
		"8080": 8080, // nextcloud, vaultwarden, generic
		"8096": 8096, // jellyfin
		"80":   80,   // wordpress/nginx
	}

	for port, portNum := range portMap {
		if strings.Contains(composeContent, fmt.Sprintf("- \"%s:", port)) ||
			strings.Contains(composeContent, fmt.Sprintf("- '%s:", port)) {
			return portNum
		}
	}

	return 0
}

// copyDir copies all files from src directory to dst directory
func copyDir(src, dst string) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := os.MkdirAll(dstPath, 0755); err != nil {
				return err
			}
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			data, err := os.ReadFile(srcPath)
			if err != nil {
				return err
			}
			if err := os.WriteFile(dstPath, data, 0644); err != nil {
				return err
			}
		}
	}

	return nil
}
