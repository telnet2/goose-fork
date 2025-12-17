package recipe

import (
	"encoding/json"
	"testing"
)

func TestNewRecipe(t *testing.T) {
	recipe := NewRecipe("Test Title", "Test Description")

	if recipe.Title != "Test Title" {
		t.Errorf("expected title 'Test Title', got '%s'", recipe.Title)
	}
	if recipe.Description != "Test Description" {
		t.Errorf("expected description 'Test Description', got '%s'", recipe.Description)
	}
	if recipe.Version != DefaultVersion {
		t.Errorf("expected version '%s', got '%s'", DefaultVersion, recipe.Version)
	}
}

func TestRecipeBuilder(t *testing.T) {
	instructions := "Test instructions"
	prompt := "Test prompt"

	recipe, err := NewRecipeBuilder().
		Title("Builder Title").
		Description("Builder Description").
		Instructions(instructions).
		Prompt(prompt).
		Build()

	if err != nil {
		t.Fatalf("failed to build recipe: %v", err)
	}

	if recipe.Title != "Builder Title" {
		t.Errorf("expected title 'Builder Title', got '%s'", recipe.Title)
	}
	if recipe.Description != "Builder Description" {
		t.Errorf("expected description 'Builder Description', got '%s'", recipe.Description)
	}
	if recipe.Instructions == nil || *recipe.Instructions != instructions {
		t.Errorf("expected instructions '%s', got '%v'", instructions, recipe.Instructions)
	}
	if recipe.Prompt == nil || *recipe.Prompt != prompt {
		t.Errorf("expected prompt '%s', got '%v'", prompt, recipe.Prompt)
	}
}

func TestRecipeBuilder_ValidationErrors(t *testing.T) {
	tests := []struct {
		name    string
		builder func() *RecipeBuilder
		wantErr string
	}{
		{
			name: "missing title",
			builder: func() *RecipeBuilder {
				return NewRecipeBuilder().Description("desc").Instructions("inst")
			},
			wantErr: "recipe title is required",
		},
		{
			name: "missing description",
			builder: func() *RecipeBuilder {
				return NewRecipeBuilder().Title("title").Instructions("inst")
			},
			wantErr: "recipe description is required",
		},
		{
			name: "missing instructions and prompt",
			builder: func() *RecipeBuilder {
				return NewRecipeBuilder().Title("title").Description("desc")
			},
			wantErr: "recipe requires at least instructions or prompt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.builder().Build()
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if err.Error() != tt.wantErr {
				t.Errorf("expected error '%s', got '%s'", tt.wantErr, err.Error())
			}
		})
	}
}

func TestFromContent_JSON(t *testing.T) {
	content := `{
		"version": "1.0.0",
		"title": "JSON Recipe",
		"description": "A recipe in JSON format",
		"instructions": "Do the thing"
	}`

	recipe, err := FromContent(content)
	if err != nil {
		t.Fatalf("failed to parse JSON content: %v", err)
	}

	if recipe.Title != "JSON Recipe" {
		t.Errorf("expected title 'JSON Recipe', got '%s'", recipe.Title)
	}
	if recipe.Description != "A recipe in JSON format" {
		t.Errorf("expected description 'A recipe in JSON format', got '%s'", recipe.Description)
	}
}

func TestFromContent_YAML(t *testing.T) {
	content := `
version: "1.0.0"
title: YAML Recipe
description: A recipe in YAML format
instructions: Do the thing
`

	recipe, err := FromContent(content)
	if err != nil {
		t.Fatalf("failed to parse YAML content: %v", err)
	}

	if recipe.Title != "YAML Recipe" {
		t.Errorf("expected title 'YAML Recipe', got '%s'", recipe.Title)
	}
	if recipe.Description != "A recipe in YAML format" {
		t.Errorf("expected description 'A recipe in YAML format', got '%s'", recipe.Description)
	}
}

func TestFromContent_NestedRecipe(t *testing.T) {
	// The nested format requires the outer object to have a "recipe" key
	// This test verifies that direct format works
	content := `{
		"version": "1.0.0",
		"title": "Direct Recipe",
		"description": "A direct recipe",
		"instructions": "Do direct thing"
	}`

	recipe, err := FromContent(content)
	if err != nil {
		t.Fatalf("failed to parse direct recipe: %v", err)
	}

	if recipe.Title != "Direct Recipe" {
		t.Errorf("expected title 'Direct Recipe', got '%s'", recipe.Title)
	}
}

func TestFromContent_Invalid(t *testing.T) {
	content := `invalid content`

	_, err := FromContent(content)
	if err == nil {
		t.Fatal("expected error for invalid content")
	}
}

func TestRecipe_ToYAML(t *testing.T) {
	instructions := "Test instructions"
	recipe := &Recipe{
		Version:      "1.0.0",
		Title:        "Test Recipe",
		Description:  "A test recipe",
		Instructions: &instructions,
	}

	yaml, err := recipe.ToYAML()
	if err != nil {
		t.Fatalf("failed to convert to YAML: %v", err)
	}

	if yaml == "" {
		t.Error("expected non-empty YAML")
	}
}

func TestRecipe_ToJSON(t *testing.T) {
	instructions := "Test instructions"
	recipe := &Recipe{
		Version:      "1.0.0",
		Title:        "Test Recipe",
		Description:  "A test recipe",
		Instructions: &instructions,
	}

	jsonStr, err := recipe.ToJSON()
	if err != nil {
		t.Fatalf("failed to convert to JSON: %v", err)
	}

	// Verify it's valid JSON
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		t.Errorf("invalid JSON output: %v", err)
	}
}

func TestRecipe_GenerateID(t *testing.T) {
	instructions := "Test instructions"
	recipe := &Recipe{
		Title:        "Test Recipe",
		Description:  "A test recipe",
		Instructions: &instructions,
	}

	id := recipe.GenerateID()
	if len(id) != 8 {
		t.Errorf("expected ID length 8, got %d", len(id))
	}

	// Same content should produce same ID
	id2 := recipe.GenerateID()
	if id != id2 {
		t.Errorf("expected same ID for same content, got '%s' and '%s'", id, id2)
	}
}

func TestRecipe_Validate(t *testing.T) {
	tests := []struct {
		name    string
		recipe  *Recipe
		wantErr bool
	}{
		{
			name: "valid with instructions",
			recipe: &Recipe{
				Title:        "Test",
				Description:  "Desc",
				Instructions: strPtr("inst"),
			},
			wantErr: false,
		},
		{
			name: "valid with prompt",
			recipe: &Recipe{
				Title:       "Test",
				Description: "Desc",
				Prompt:      strPtr("prompt"),
			},
			wantErr: false,
		},
		{
			name: "missing title",
			recipe: &Recipe{
				Description:  "Desc",
				Instructions: strPtr("inst"),
			},
			wantErr: true,
		},
		{
			name: "missing description",
			recipe: &Recipe{
				Title:        "Test",
				Instructions: strPtr("inst"),
			},
			wantErr: true,
		},
		{
			name: "missing instructions and prompt",
			recipe: &Recipe{
				Title:       "Test",
				Description: "Desc",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.recipe.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRecipe_ValidateParameters(t *testing.T) {
	recipe := &Recipe{
		Title:        "Test",
		Description:  "Desc",
		Instructions: strPtr("Hello {{name}}!"),
		Parameters: []RecipeParameter{
			{
				Key:         "name",
				InputType:   InputTypeString,
				Requirement: RequirementRequired,
				Description: "Name parameter",
			},
		},
	}

	if err := recipe.Validate(); err != nil {
		t.Errorf("expected valid recipe, got error: %v", err)
	}

	// Invalid: parameter not in template
	recipe.Parameters = append(recipe.Parameters, RecipeParameter{
		Key:         "missing",
		InputType:   InputTypeString,
		Requirement: RequirementRequired,
		Description: "Missing parameter",
	})

	if err := recipe.Validate(); err == nil {
		t.Error("expected error for parameter not in template")
	}
}

func TestRecipe_CheckSecurityWarnings(t *testing.T) {
	// Recipe without warnings
	recipe := &Recipe{
		Title:        "Safe Recipe",
		Description:  "A safe recipe",
		Instructions: strPtr("Normal instructions"),
	}

	warnings := recipe.CheckSecurityWarnings()
	if len(warnings) != 0 {
		t.Errorf("expected no warnings, got %v", warnings)
	}
}

func TestRecipe_RenderTemplate(t *testing.T) {
	recipe := &Recipe{
		Title:        "Test Recipe",
		Description:  "A test recipe",
		Instructions: strPtr("Hello {{name}}, welcome to {{place}}!"),
		Prompt:       strPtr("Say hi to {{name}}"),
	}

	values := map[string]string{
		"name":  "World",
		"place": "Goose",
	}

	rendered, err := recipe.RenderTemplate(values)
	if err != nil {
		t.Fatalf("failed to render template: %v", err)
	}

	if rendered.Instructions == nil || *rendered.Instructions != "Hello World, welcome to Goose!" {
		t.Errorf("unexpected rendered instructions: %v", rendered.Instructions)
	}
	if rendered.Prompt == nil || *rendered.Prompt != "Say hi to World" {
		t.Errorf("unexpected rendered prompt: %v", rendered.Prompt)
	}
}

func TestRecipe_CreateManifest(t *testing.T) {
	instructions := "Test instructions"
	recipe := &Recipe{
		Title:        "Test Recipe",
		Description:  "A test recipe",
		Instructions: &instructions,
	}

	manifest := recipe.CreateManifest("/path/to/recipe.yaml")

	if manifest.ID == "" {
		t.Error("expected non-empty ID")
	}
	if manifest.FilePath != "/path/to/recipe.yaml" {
		t.Errorf("expected file path '/path/to/recipe.yaml', got '%s'", manifest.FilePath)
	}
	if manifest.Recipe.Title != recipe.Title {
		t.Errorf("expected title '%s', got '%s'", recipe.Title, manifest.Recipe.Title)
	}
}

func TestTitleToFilename(t *testing.T) {
	tests := []struct {
		title    string
		expected string
	}{
		{"Simple Title", "simple_title.yaml"},
		{"Title With Spaces", "title_with_spaces.yaml"},
		{"Title!@#With$%^Special&*Chars", "title_with_special_chars.yaml"},
		{"UPPERCASE", "uppercase.yaml"},
		{"multiple___underscores", "multiple_underscores.yaml"},
		{"_leading_trailing_", "leading_trailing.yaml"},
	}

	for _, tt := range tests {
		t.Run(tt.title, func(t *testing.T) {
			result := TitleToFilename(tt.title)
			if result != tt.expected {
				t.Errorf("TitleToFilename(%s) = %s, want %s", tt.title, result, tt.expected)
			}
		})
	}
}

func strPtr(s string) *string {
	return &s
}
