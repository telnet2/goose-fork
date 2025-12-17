package recipe

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

// Storage manages recipe file storage
type Storage struct {
	globalDir string
	localDir  string
	mu        sync.RWMutex
}

// NewStorage creates a new recipe storage
func NewStorage(dataDir string) *Storage {
	return &Storage{
		globalDir: filepath.Join(dataDir, "recipes"),
		localDir:  ".goose/recipes",
	}
}

// GlobalRecipesDir returns the global recipes directory
func GlobalRecipesDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}

	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(homeDir, "Library", "Application Support", "goose", "recipes")
	case "windows":
		appData := os.Getenv("APPDATA")
		if appData == "" {
			appData = filepath.Join(homeDir, "AppData", "Roaming")
		}
		return filepath.Join(appData, "goose", "recipes")
	default:
		xdgConfig := os.Getenv("XDG_CONFIG_HOME")
		if xdgConfig == "" {
			xdgConfig = filepath.Join(homeDir, ".config")
		}
		return filepath.Join(xdgConfig, "goose", "recipes")
	}
}

// GetSearchPaths returns all paths to search for recipes
func (s *Storage) GetSearchPaths() []string {
	paths := []string{"."}

	// Add paths from environment variable
	if envPaths := os.Getenv("GOOSE_RECIPE_PATH"); envPaths != "" {
		sep := ":"
		if runtime.GOOS == "windows" {
			sep = ";"
		}
		paths = append(paths, strings.Split(envPaths, sep)...)
	}

	// Add global directory
	paths = append(paths, s.globalDir)

	// Add local directory
	paths = append(paths, s.localDir)

	return paths
}

// List lists all recipes from all search paths
func (s *Storage) List() ([]RecipeManifest, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var manifests []RecipeManifest
	seen := make(map[string]bool) // Track by ID to avoid duplicates

	for _, searchPath := range s.GetSearchPaths() {
		recipes, err := s.listFromPath(searchPath)
		if err != nil {
			continue // Skip paths that don't exist or can't be read
		}

		for _, manifest := range recipes {
			if !seen[manifest.ID] {
				seen[manifest.ID] = true
				manifests = append(manifests, manifest)
			}
		}
	}

	return manifests, nil
}

// listFromPath lists recipes from a specific path
func (s *Storage) listFromPath(searchPath string) ([]RecipeManifest, error) {
	var manifests []RecipeManifest

	// Check if path exists
	info, err := os.Stat(searchPath)
	if err != nil {
		return nil, err
	}

	if info.IsDir() {
		// Walk directory for recipe files
		err := filepath.Walk(searchPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if info.IsDir() {
				return nil
			}

			ext := strings.ToLower(filepath.Ext(path))
			if ext == ".yaml" || ext == ".yml" || ext == ".json" {
				recipe, err := FromFile(path)
				if err != nil {
					return nil // Skip invalid files
				}
				manifests = append(manifests, recipe.CreateManifest(path))
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	} else {
		// Single file
		recipe, err := FromFile(searchPath)
		if err != nil {
			return nil, err
		}
		manifests = append(manifests, recipe.CreateManifest(searchPath))
	}

	return manifests, nil
}

// Save saves a recipe to a file
func (s *Storage) Save(recipe *Recipe, filePath *string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var path string
	if filePath != nil && *filePath != "" {
		path = *filePath
	} else {
		// Generate filename from title
		filename := TitleToFilename(recipe.Title)
		path = filepath.Join(s.globalDir, filename)

		// Handle naming conflicts
		basePath := path
		ext := filepath.Ext(path)
		nameWithoutExt := strings.TrimSuffix(basePath, ext)
		counter := 1
		for {
			if _, err := os.Stat(path); os.IsNotExist(err) {
				break
			}
			path = fmt.Sprintf("%s_%d%s", nameWithoutExt, counter, ext)
			counter++
		}
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	// Convert to YAML
	content, err := recipe.ToYAML()
	if err != nil {
		return "", fmt.Errorf("failed to serialize recipe: %w", err)
	}

	// Write file
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	return path, nil
}

// Delete deletes a recipe by ID
func (s *Storage) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Find the recipe file
	manifests, err := s.List()
	if err != nil {
		return err
	}

	for _, manifest := range manifests {
		if manifest.ID == id {
			return os.Remove(manifest.FilePath)
		}
	}

	return fmt.Errorf("recipe not found: %s", id)
}

// Get retrieves a recipe by ID
func (s *Storage) Get(id string) (*RecipeManifest, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	manifests, err := s.List()
	if err != nil {
		return nil, err
	}

	for _, manifest := range manifests {
		if manifest.ID == id {
			return &manifest, nil
		}
	}

	return nil, fmt.Errorf("recipe not found: %s", id)
}

// GetByFilePath retrieves a recipe by file path
func (s *Storage) GetByFilePath(path string) (*RecipeManifest, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	recipe, err := FromFile(path)
	if err != nil {
		return nil, err
	}

	manifest := recipe.CreateManifest(path)
	return &manifest, nil
}

// FindRecipeFile searches for a recipe file by name or path
func (s *Storage) FindRecipeFile(nameOrPath string) (string, error) {
	// Check if it's an absolute path
	if filepath.IsAbs(nameOrPath) {
		if _, err := os.Stat(nameOrPath); err == nil {
			return nameOrPath, nil
		}
	}

	// Search in all paths
	for _, searchPath := range s.GetSearchPaths() {
		// Try direct path
		candidate := filepath.Join(searchPath, nameOrPath)
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}

		// Try with .yaml extension
		candidate = filepath.Join(searchPath, nameOrPath+".yaml")
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}

		// Try with .yml extension
		candidate = filepath.Join(searchPath, nameOrPath+".yml")
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}

		// Try with .json extension
		candidate = filepath.Join(searchPath, nameOrPath+".json")
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}

	return "", fmt.Errorf("recipe not found: %s", nameOrPath)
}

// UpdateScheduleCron updates the schedule cron for a recipe
func (s *Storage) UpdateScheduleCron(id string, cron *string) error {
	// This is typically stored in the scheduler, not the recipe file itself
	// The manifest's ScheduleCron is populated from the scheduler state
	return nil
}

// UpdateSlashCommand updates the slash command for a recipe
func (s *Storage) UpdateSlashCommand(id string, command *string) error {
	// This would typically be stored in config, not the recipe file
	return nil
}

// CopyToScheduledRecipes copies a recipe file to the scheduled recipes directory
func (s *Storage) CopyToScheduledRecipes(sourcePath string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	scheduledDir := filepath.Join(s.globalDir, "..", "scheduled_recipes")
	if err := os.MkdirAll(scheduledDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create scheduled recipes directory: %w", err)
	}

	filename := filepath.Base(sourcePath)
	destPath := filepath.Join(scheduledDir, filename)

	// Handle naming conflicts
	basePath := destPath
	ext := filepath.Ext(destPath)
	nameWithoutExt := strings.TrimSuffix(basePath, ext)
	counter := 1
	for {
		if _, err := os.Stat(destPath); os.IsNotExist(err) {
			break
		}
		destPath = fmt.Sprintf("%s_%d%s", nameWithoutExt, counter, ext)
		counter++
	}

	// Read source
	content, err := os.ReadFile(sourcePath)
	if err != nil {
		return "", fmt.Errorf("failed to read source file: %w", err)
	}

	// Write to destination
	if err := os.WriteFile(destPath, content, 0644); err != nil {
		return "", fmt.Errorf("failed to write destination file: %w", err)
	}

	return destPath, nil
}

// EnsureDirectoriesExist creates the recipe directories if they don't exist
func (s *Storage) EnsureDirectoriesExist() error {
	if err := os.MkdirAll(s.globalDir, 0755); err != nil {
		return err
	}
	return nil
}
