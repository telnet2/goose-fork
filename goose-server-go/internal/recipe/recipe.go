package recipe

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Recipe represents a goose recipe
type Recipe struct {
	Version      string             `json:"version" yaml:"version"`
	Title        string             `json:"title" yaml:"title"`
	Description  string             `json:"description" yaml:"description"`
	Instructions *string            `json:"instructions,omitempty" yaml:"instructions,omitempty"`
	Prompt       *string            `json:"prompt,omitempty" yaml:"prompt,omitempty"`
	Extensions   []ExtensionRef     `json:"extensions,omitempty" yaml:"extensions,omitempty"`
	Settings     *Settings          `json:"settings,omitempty" yaml:"settings,omitempty"`
	Activities   []string           `json:"activities,omitempty" yaml:"activities,omitempty"`
	Author       *Author            `json:"author,omitempty" yaml:"author,omitempty"`
	Parameters   []RecipeParameter  `json:"parameters,omitempty" yaml:"parameters,omitempty"`
	Response     *Response          `json:"response,omitempty" yaml:"response,omitempty"`
	SubRecipes   []SubRecipe        `json:"sub_recipes,omitempty" yaml:"sub_recipes,omitempty"`
	Retry        *RetryConfig       `json:"retry,omitempty" yaml:"retry,omitempty"`
}

// ExtensionRef references an extension in a recipe
type ExtensionRef struct {
	Type        string            `json:"type" yaml:"type"`
	Name        string            `json:"name" yaml:"name"`
	Description string            `json:"description,omitempty" yaml:"description,omitempty"`
	URI         string            `json:"uri,omitempty" yaml:"uri,omitempty"`
	Cmd         string            `json:"cmd,omitempty" yaml:"cmd,omitempty"`
	Args        []string          `json:"args,omitempty" yaml:"args,omitempty"`
	Envs        map[string]string `json:"envs,omitempty" yaml:"envs,omitempty"`
	EnvKeys     []string          `json:"env_keys,omitempty" yaml:"env_keys,omitempty"`
	Timeout     *uint64           `json:"timeout,omitempty" yaml:"timeout,omitempty"`
}

// Settings contains recipe settings
type Settings struct {
	GooseProvider *string  `json:"goose_provider,omitempty" yaml:"goose_provider,omitempty"`
	GooseModel    *string  `json:"goose_model,omitempty" yaml:"goose_model,omitempty"`
	Temperature   *float32 `json:"temperature,omitempty" yaml:"temperature,omitempty"`
}

// Author represents recipe author information
type Author struct {
	Contact  *string `json:"contact,omitempty" yaml:"contact,omitempty"`
	Metadata *string `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}

// RecipeParameter represents a parameter in a recipe
type RecipeParameter struct {
	Key         string                      `json:"key" yaml:"key"`
	InputType   RecipeParameterInputType    `json:"input_type" yaml:"input_type"`
	Requirement RecipeParameterRequirement  `json:"requirement" yaml:"requirement"`
	Description string                      `json:"description" yaml:"description"`
	Default     *string                     `json:"default,omitempty" yaml:"default,omitempty"`
	Options     []string                    `json:"options,omitempty" yaml:"options,omitempty"`
}

// RecipeParameterInputType defines the input type for a parameter
type RecipeParameterInputType string

const (
	InputTypeString  RecipeParameterInputType = "string"
	InputTypeNumber  RecipeParameterInputType = "number"
	InputTypeBoolean RecipeParameterInputType = "boolean"
	InputTypeDate    RecipeParameterInputType = "date"
	InputTypeFile    RecipeParameterInputType = "file"
	InputTypeSelect  RecipeParameterInputType = "select"
)

// RecipeParameterRequirement defines if a parameter is required
type RecipeParameterRequirement string

const (
	RequirementRequired   RecipeParameterRequirement = "required"
	RequirementOptional   RecipeParameterRequirement = "optional"
	RequirementUserPrompt RecipeParameterRequirement = "user_prompt"
)

// Response defines the expected response format
type Response struct {
	JSONSchema *json.RawMessage `json:"json_schema,omitempty" yaml:"json_schema,omitempty"`
}

// SubRecipe represents a nested recipe
type SubRecipe struct {
	Name                    string            `json:"name" yaml:"name"`
	Path                    string            `json:"path" yaml:"path"`
	Values                  map[string]string `json:"values,omitempty" yaml:"values,omitempty"`
	SequentialWhenRepeated  bool              `json:"sequential_when_repeated,omitempty" yaml:"sequential_when_repeated,omitempty"`
	Description             *string           `json:"description,omitempty" yaml:"description,omitempty"`
}

// RetryConfig defines retry behavior
type RetryConfig struct {
	MaxRetries int    `json:"max_retries" yaml:"max_retries"`
	Delay      string `json:"delay,omitempty" yaml:"delay,omitempty"`
}

// RecipeManifest contains metadata about a recipe file
type RecipeManifest struct {
	ID           string  `json:"id"`
	Recipe       Recipe  `json:"recipe"`
	FilePath     string  `json:"file_path"`
	LastModified string  `json:"last_modified"`
	ScheduleCron *string `json:"schedule_cron,omitempty"`
	SlashCommand *string `json:"slash_command,omitempty"`
}

// DefaultVersion is the default recipe version
const DefaultVersion = "1.0.0"

// NewRecipe creates a new recipe with defaults
func NewRecipe(title, description string) *Recipe {
	return &Recipe{
		Version:     DefaultVersion,
		Title:       title,
		Description: description,
	}
}

// RecipeBuilder provides a fluent API for building recipes
type RecipeBuilder struct {
	recipe Recipe
}

// NewRecipeBuilder creates a new recipe builder
func NewRecipeBuilder() *RecipeBuilder {
	return &RecipeBuilder{
		recipe: Recipe{
			Version: DefaultVersion,
		},
	}
}

// Title sets the recipe title
func (b *RecipeBuilder) Title(title string) *RecipeBuilder {
	b.recipe.Title = title
	return b
}

// Description sets the recipe description
func (b *RecipeBuilder) Description(desc string) *RecipeBuilder {
	b.recipe.Description = desc
	return b
}

// Instructions sets the recipe instructions
func (b *RecipeBuilder) Instructions(instructions string) *RecipeBuilder {
	b.recipe.Instructions = &instructions
	return b
}

// Prompt sets the recipe prompt
func (b *RecipeBuilder) Prompt(prompt string) *RecipeBuilder {
	b.recipe.Prompt = &prompt
	return b
}

// Settings sets the recipe settings
func (b *RecipeBuilder) Settings(settings *Settings) *RecipeBuilder {
	b.recipe.Settings = settings
	return b
}

// Extensions sets the recipe extensions
func (b *RecipeBuilder) Extensions(exts []ExtensionRef) *RecipeBuilder {
	b.recipe.Extensions = exts
	return b
}

// Parameters sets the recipe parameters
func (b *RecipeBuilder) Parameters(params []RecipeParameter) *RecipeBuilder {
	b.recipe.Parameters = params
	return b
}

// Author sets the recipe author
func (b *RecipeBuilder) Author(author *Author) *RecipeBuilder {
	b.recipe.Author = author
	return b
}

// Build validates and returns the recipe
func (b *RecipeBuilder) Build() (*Recipe, error) {
	if b.recipe.Title == "" {
		return nil, fmt.Errorf("recipe title is required")
	}
	if b.recipe.Description == "" {
		return nil, fmt.Errorf("recipe description is required")
	}
	if b.recipe.Instructions == nil && b.recipe.Prompt == nil {
		return nil, fmt.Errorf("recipe requires at least instructions or prompt")
	}
	return &b.recipe, nil
}

// FromContent parses a recipe from YAML or JSON content
func FromContent(content string) (*Recipe, error) {
	content = strings.TrimSpace(content)

	// Try to detect format
	var recipe Recipe
	var outerRecipe struct {
		Recipe Recipe `json:"recipe" yaml:"recipe"`
	}

	// Check if content starts with { (JSON) or not (YAML)
	if strings.HasPrefix(content, "{") {
		// Try JSON
		if err := json.Unmarshal([]byte(content), &recipe); err == nil {
			return &recipe, nil
		}
		// Try nested recipe format
		if err := json.Unmarshal([]byte(content), &outerRecipe); err == nil && outerRecipe.Recipe.Title != "" {
			return &outerRecipe.Recipe, nil
		}
		return nil, fmt.Errorf("invalid JSON recipe format")
	}

	// Try YAML
	if err := yaml.Unmarshal([]byte(content), &recipe); err == nil && recipe.Title != "" {
		return &recipe, nil
	}
	// Try nested recipe format
	if err := yaml.Unmarshal([]byte(content), &outerRecipe); err == nil && outerRecipe.Recipe.Title != "" {
		return &outerRecipe.Recipe, nil
	}

	return nil, fmt.Errorf("invalid recipe format")
}

// FromFile loads a recipe from a file
func FromFile(path string) (*Recipe, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read recipe file: %w", err)
	}
	return FromContent(string(content))
}

// ToYAML converts the recipe to YAML format
func (r *Recipe) ToYAML() (string, error) {
	data, err := yaml.Marshal(r)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// ToJSON converts the recipe to JSON format
func (r *Recipe) ToJSON() (string, error) {
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// GenerateID generates a short hash-based ID for a recipe
func (r *Recipe) GenerateID() string {
	content := r.Title + r.Description
	if r.Instructions != nil {
		content += *r.Instructions
	}
	if r.Prompt != nil {
		content += *r.Prompt
	}

	hash := sha256.Sum256([]byte(content))
	return hex.EncodeToString(hash[:])[:8]
}

// Validate validates the recipe
func (r *Recipe) Validate() error {
	if r.Title == "" {
		return fmt.Errorf("recipe title is required")
	}
	if r.Description == "" {
		return fmt.Errorf("recipe description is required")
	}
	if r.Instructions == nil && r.Prompt == nil {
		return fmt.Errorf("recipe requires at least instructions or prompt")
	}

	// Validate parameters exist in template if present
	if r.Parameters != nil {
		templateContent := ""
		if r.Instructions != nil {
			templateContent += *r.Instructions
		}
		if r.Prompt != nil {
			templateContent += *r.Prompt
		}

		for _, param := range r.Parameters {
			placeholder := "{{" + param.Key + "}}"
			if !strings.Contains(templateContent, placeholder) {
				return fmt.Errorf("parameter %s not found in template", param.Key)
			}
		}
	}

	return nil
}

// CheckSecurityWarnings checks for potential security issues in the recipe
func (r *Recipe) CheckSecurityWarnings() []string {
	var warnings []string

	// Check for hidden unicode tags
	hiddenTagPattern := regexp.MustCompile(`[\x{E0000}-\x{E007F}]`)

	checkString := func(name, value string) {
		if hiddenTagPattern.MatchString(value) {
			warnings = append(warnings, fmt.Sprintf("Hidden unicode tags found in %s", name))
		}
	}

	if r.Instructions != nil {
		checkString("instructions", *r.Instructions)
	}
	if r.Prompt != nil {
		checkString("prompt", *r.Prompt)
	}
	for i, activity := range r.Activities {
		checkString(fmt.Sprintf("activity[%d]", i), activity)
	}

	return warnings
}

// RenderTemplate renders the recipe with given parameter values
func (r *Recipe) RenderTemplate(values map[string]string) (*Recipe, error) {
	rendered := *r // Copy recipe

	render := func(s *string) *string {
		if s == nil {
			return nil
		}
		result := *s
		for key, value := range values {
			placeholder := "{{" + key + "}}"
			result = strings.ReplaceAll(result, placeholder, value)
		}
		return &result
	}

	rendered.Instructions = render(r.Instructions)
	rendered.Prompt = render(r.Prompt)

	return &rendered, nil
}

// CreateManifest creates a manifest for this recipe
func (r *Recipe) CreateManifest(filePath string) RecipeManifest {
	info, _ := os.Stat(filePath)
	lastModified := time.Now()
	if info != nil {
		lastModified = info.ModTime()
	}

	return RecipeManifest{
		ID:           r.GenerateID(),
		Recipe:       *r,
		FilePath:     filePath,
		LastModified: lastModified.Format(time.RFC3339),
	}
}

// TitleToFilename converts a recipe title to a safe filename
func TitleToFilename(title string) string {
	// Replace spaces and special characters with underscores
	safe := regexp.MustCompile(`[^a-zA-Z0-9_-]`).ReplaceAllString(title, "_")
	// Remove consecutive underscores
	safe = regexp.MustCompile(`_+`).ReplaceAllString(safe, "_")
	// Trim underscores
	safe = strings.Trim(safe, "_")
	// Lowercase
	safe = strings.ToLower(safe)
	// Add extension
	return safe + ".yaml"
}
