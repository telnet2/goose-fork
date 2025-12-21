package recipe

import (
	"testing"
)

func TestEncodeDecode(t *testing.T) {
	original := &Recipe{
		Version:      "1.0.0",
		Title:        "Test Recipe",
		Description:  "A test recipe for encoding",
		Instructions: strPtr("Follow these instructions"),
		Prompt:       strPtr("This is the prompt"),
	}

	// Encode
	encoded, err := Encode(original)
	if err != nil {
		t.Fatalf("failed to encode recipe: %v", err)
	}

	if encoded == "" {
		t.Fatal("encoded string should not be empty")
	}

	// Decode
	decoded, err := Decode(encoded)
	if err != nil {
		t.Fatalf("failed to decode recipe: %v", err)
	}

	// Compare
	if decoded.Title != original.Title {
		t.Errorf("title mismatch: got '%s', want '%s'", decoded.Title, original.Title)
	}
	if decoded.Description != original.Description {
		t.Errorf("description mismatch: got '%s', want '%s'", decoded.Description, original.Description)
	}
	if decoded.Instructions == nil || *decoded.Instructions != *original.Instructions {
		t.Errorf("instructions mismatch")
	}
	if decoded.Prompt == nil || *decoded.Prompt != *original.Prompt {
		t.Errorf("prompt mismatch")
	}
}

func TestEncode_ComplexRecipe(t *testing.T) {
	original := &Recipe{
		Version:      "1.0.0",
		Title:        "Complex Recipe",
		Description:  "A complex recipe with parameters",
		Instructions: strPtr("Use parameter {{param1}} and {{param2}}"),
		Parameters: []RecipeParameter{
			{
				Key:         "param1",
				InputType:   InputTypeString,
				Requirement: RequirementRequired,
				Description: "First parameter",
			},
			{
				Key:         "param2",
				InputType:   InputTypeNumber,
				Requirement: RequirementOptional,
				Description: "Second parameter",
				Default:     strPtr("42"),
			},
		},
		Extensions: []ExtensionRef{
			{
				Type: "builtin",
				Name: "test-extension",
			},
		},
	}

	encoded, err := Encode(original)
	if err != nil {
		t.Fatalf("failed to encode complex recipe: %v", err)
	}

	decoded, err := Decode(encoded)
	if err != nil {
		t.Fatalf("failed to decode complex recipe: %v", err)
	}

	if len(decoded.Parameters) != 2 {
		t.Errorf("expected 2 parameters, got %d", len(decoded.Parameters))
	}

	if len(decoded.Extensions) != 1 {
		t.Errorf("expected 1 extension, got %d", len(decoded.Extensions))
	}
}

func TestDecode_InvalidInput(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"empty string", ""},
		{"invalid base64", "!!!not-base64!!!"},
		{"valid base64 but not gzip", "aGVsbG8gd29ybGQ="}, // "hello world" in base64
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Decode(tt.input)
			if err == nil {
				t.Errorf("expected error for input '%s'", tt.input)
			}
		})
	}
}

func TestEncode_EmptyRecipe(t *testing.T) {
	recipe := &Recipe{}

	encoded, err := Encode(recipe)
	if err != nil {
		t.Fatalf("failed to encode empty recipe: %v", err)
	}

	// Decoding an empty recipe should fail because title is required
	_, err = Decode(encoded)
	if err == nil {
		t.Fatal("expected error decoding empty recipe (no title)")
	}
}

func TestEncode_UnicodeContent(t *testing.T) {
	original := &Recipe{
		Version:      "1.0.0",
		Title:        "Unicode Recipe 日本語",
		Description:  "Description with émojis 🎉",
		Instructions: strPtr("Instructions in 中文"),
	}

	encoded, err := Encode(original)
	if err != nil {
		t.Fatalf("failed to encode unicode recipe: %v", err)
	}

	decoded, err := Decode(encoded)
	if err != nil {
		t.Fatalf("failed to decode unicode recipe: %v", err)
	}

	if decoded.Title != original.Title {
		t.Errorf("title mismatch: got '%s', want '%s'", decoded.Title, original.Title)
	}
	if decoded.Description != original.Description {
		t.Errorf("description mismatch")
	}
}

func TestEncode_LargeRecipe(t *testing.T) {
	// Create a recipe with large content
	largeContent := ""
	for i := 0; i < 1000; i++ {
		largeContent += "This is a line of content. "
	}

	original := &Recipe{
		Version:      "1.0.0",
		Title:        "Large Recipe",
		Description:  "A recipe with large content",
		Instructions: &largeContent,
	}

	encoded, err := Encode(original)
	if err != nil {
		t.Fatalf("failed to encode large recipe: %v", err)
	}

	// Compression should make the encoded string smaller than the original
	if len(encoded) >= len(largeContent) {
		t.Logf("warning: encoded size (%d) >= original size (%d), compression may not be effective", len(encoded), len(largeContent))
	}

	decoded, err := Decode(encoded)
	if err != nil {
		t.Fatalf("failed to decode large recipe: %v", err)
	}

	if decoded.Instructions == nil || *decoded.Instructions != largeContent {
		t.Error("instructions content mismatch")
	}
}
