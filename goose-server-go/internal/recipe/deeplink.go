package recipe

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// DeeplinkPrefix is the URL prefix for recipe deeplinks
const DeeplinkPrefix = "goose://recipe/"

// Encode encodes a recipe to a deeplink string
func Encode(recipe *Recipe) (string, error) {
	// Convert to JSON
	jsonData, err := json.Marshal(recipe)
	if err != nil {
		return "", fmt.Errorf("failed to marshal recipe: %w", err)
	}

	// Compress with gzip
	var compressed bytes.Buffer
	gzWriter := gzip.NewWriter(&compressed)
	if _, err := gzWriter.Write(jsonData); err != nil {
		return "", fmt.Errorf("failed to compress recipe: %w", err)
	}
	if err := gzWriter.Close(); err != nil {
		return "", fmt.Errorf("failed to close gzip writer: %w", err)
	}

	// Base64 encode
	encoded := base64.URLEncoding.EncodeToString(compressed.Bytes())

	// Add prefix
	return DeeplinkPrefix + encoded, nil
}

// Decode decodes a recipe from a deeplink string
func Decode(link string) (*Recipe, error) {
	// Remove prefix if present
	encoded := link
	if strings.HasPrefix(link, DeeplinkPrefix) {
		encoded = strings.TrimPrefix(link, DeeplinkPrefix)
	}

	// Base64 decode
	compressed, err := base64.URLEncoding.DecodeString(encoded)
	if err != nil {
		// Try standard base64 (without URL-safe encoding)
		compressed, err = base64.StdEncoding.DecodeString(encoded)
		if err != nil {
			return nil, fmt.Errorf("failed to decode base64: %w", err)
		}
	}

	// Try to decompress with gzip
	var jsonData []byte
	gzReader, err := gzip.NewReader(bytes.NewReader(compressed))
	if err != nil {
		// If gzip fails, try treating as raw JSON
		jsonData = compressed
	} else {
		jsonData, err = io.ReadAll(gzReader)
		gzReader.Close()
		if err != nil {
			return nil, fmt.Errorf("failed to decompress recipe: %w", err)
		}
	}

	// Parse JSON
	var recipe Recipe
	if err := json.Unmarshal(jsonData, &recipe); err != nil {
		return nil, fmt.Errorf("failed to parse recipe JSON: %w", err)
	}

	// Validate
	if recipe.Title == "" {
		return nil, fmt.Errorf("decoded recipe has no title")
	}

	return &recipe, nil
}

// IsDeeplink checks if a string is a recipe deeplink
func IsDeeplink(s string) bool {
	return strings.HasPrefix(s, DeeplinkPrefix)
}

// ExtractFromURL extracts the encoded recipe data from a full URL
func ExtractFromURL(url string) string {
	// Handle various URL formats
	if strings.HasPrefix(url, DeeplinkPrefix) {
		return strings.TrimPrefix(url, DeeplinkPrefix)
	}

	// Handle web URLs that contain the recipe
	if strings.Contains(url, "recipe=") {
		parts := strings.SplitN(url, "recipe=", 2)
		if len(parts) == 2 {
			// Remove any trailing parameters
			encoded := parts[1]
			if idx := strings.Index(encoded, "&"); idx != -1 {
				encoded = encoded[:idx]
			}
			return encoded
		}
	}

	return url
}
