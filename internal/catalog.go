package internal

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"gofr.dev/pkg/gofr"
)

// Update updates the local catalog from the repository
func (s *Service) Update(c *gofr.Context) (interface{}, error) {
	if _, err := os.Stat(s.catalogPath); os.IsNotExist(err) {
		return s.cloneCatalog()
	}

	return s.pullCatalog()
}

func (s *Service) cloneCatalog() (interface{}, error) {
	if err := os.MkdirAll(filepath.Dir(s.catalogPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create catalog directory: %w", err)
	}

	tempDir := s.catalogPath + ".tmp"
	defer os.RemoveAll(tempDir)

	cmd := exec.Command("git", "clone", "--depth", "1", catalogRepoURL, tempDir)
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to clone catalog: %w", err)
	}

	// Move catalog contents (excluding .git)
	srcCatalog := tempDir
	if err := os.Rename(srcCatalog, s.catalogPath); err != nil {
		return nil, fmt.Errorf("failed to move catalog: %w", err)
	}

	// Remove .git directory to keep it clean
	targetGitDir := filepath.Join(s.catalogPath, ".git")
	_ = os.RemoveAll(targetGitDir)

	return "\n✅ Catalog downloaded successfully!\n\nRun 'opensourcer catalog' to see available software.\n", nil
}

func (s *Service) pullCatalog() (interface{}, error) {
	if _, err := os.Stat(s.catalogPath); os.IsNotExist(err) {
		return s.cloneCatalog()
	}

	// Check if it's a git repo
	gitDir := filepath.Join(s.catalogPath, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		// Not a git repo, re-clone
		_ = os.RemoveAll(s.catalogPath)
		return s.cloneCatalog()
	}

	cmd := exec.Command("git", "-C", s.catalogPath, "pull", "--rebase")
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to update catalog: %w", err)
	}

	return "\n✅ Catalog updated successfully!\n", nil
}

// listCatalogItems returns a list of software slugs in the catalog
func (s *Service) listCatalogItems() ([]string, error) {
	entries, err := os.ReadDir(s.catalogPath)
	if err != nil {
		return nil, err
	}

	var items []string
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), "_") {
			continue
		}
		items = append(items, entry.Name())
	}

	return items, nil
}
