package docx

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
)

type Format string

const (
	JSON     Format = "json"
	Markdown Format = "markdown"
	HTML     Format = "html"
	CURL     Format = "curl"
)

type Generator struct {
	Routers []*RouterDoc
}

func NewGenerator() *Generator {
	return &Generator{
		Routers: []*RouterDoc{},
	}
}

func (g *Generator) AddRouter(router *RouterDoc) *Generator {
	g.Routers = append(g.Routers, router)
	return g
}

func (g *Generator) GenerateJSON(outputPath string) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Marshal the generator to JSON
	data, err := json.MarshalIndent(g.Routers, "", "  ")
	if err != nil {
		return err
	}

	// Write to file
	return os.WriteFile(outputPath, data, 0644)
}
func (g *Generator) GenerateCurlDocs(baseURL string, outputPath string) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Create output file
	file, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Write header
	fmt.Fprintf(file, "# API Curl Examples\n\n")

	// For each router in the generator
	for _, router := range g.Routers {
		fmt.Fprintf(file, "## %s\n\n", router.BasePath)

		// For each endpoint in the router
		for _, endpoint := range router.Endpoints {
			fmt.Fprintf(file, "### %s %s\n\n", endpoint.Method, endpoint.Path)
			fmt.Fprintf(file, "%s\n\n", endpoint.Description)

			// Generate curl command
			curlCmd := generateCurlCommand(baseURL, router.BasePath, endpoint)
			fmt.Fprintf(file, "```bash\n%s\n```\n\n", curlCmd)

			// Add response example if available
			if endpoint.ResponseExample != nil {
				respJSON, _ := json.MarshalIndent(endpoint.ResponseExample, "", "  ")
				fmt.Fprintf(file, "**Example Response:**\n\n```json\n%s\n```\n\n", string(respJSON))
			}
		}
	}

	return nil
}

func generateCurlCommand(baseURL string, basePath string, endpoint *Endpoint) string {
	// Build the base URL
	fullURL := fmt.Sprintf("%s%s%s", baseURL, basePath, endpoint.Path)

	// Add query parameters if they exist
	if len(endpoint.QueryParams) > 0 {
		queryParams := []string{}
		for _, param := range endpoint.QueryParams {
			var value string
			if param.Default != nil {
				value = fmt.Sprintf("%v", param.Default)
			} else {
				value = fmt.Sprintf("<%s>", strings.ToUpper(param.Name))
			}
			queryParams = append(queryParams, fmt.Sprintf("%s=%s", param.Name, value))
		}
		fullURL += "?" + strings.Join(queryParams, "&")
	}

	curl := fmt.Sprintf("curl -X %s \\\n  %s", endpoint.Method, fullURL)

	// Add headers
	for _, header := range endpoint.Headers {
		curl += fmt.Sprintf(" \\\n  -H '%s: %s'", header.Name, header.Value)
	}

	// Add auth if required
	if endpoint.Auth != None {
		// Add appropriate auth headers/parameters based on auth type
		switch endpoint.Auth {
		case Bearer:
			curl += " \\\n  -H 'Authorization: Bearer YOUR_TOKEN'"
		case Basic:
			curl += " \\\n  -H 'Authorization: Basic YOUR_CREDENTIALS'"
			// Add other auth types as needed
		}
	}

	// Add request body if it's a POST, PUT, PATCH
	if endpoint.Method == "POST" || endpoint.Method == "PUT" || endpoint.Method == "PATCH" {
		if endpoint.RequestExample != nil {
			bodyJSON, _ := json.Marshal(endpoint.RequestExample)
			curl += fmt.Sprintf(" \\\n  -d '%s'", string(bodyJSON))
		}
	}

	return curl
}

type SchemaField struct {
	Name     string        `json:"name"`
	Type     string        `json:"type"`
	Tag      string        `json:"tag,omitempty"`
	Required bool          `json:"required"`
	Fields   []SchemaField `json:"fields,omitempty"`
}

type Schema struct {
	Type   string        `json:"type"`
	Fields []SchemaField `json:"fields,omitempty"`
}

// Helper function to extract schema information from a type
func extractSchema(t reflect.Type) Schema {
	schema := Schema{Type: t.Name()}

	// Handle pointer types
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
		schema.Type = t.Name() + " (pointer)"
	}

	// Only extract fields for struct types
	if t.Kind() == reflect.Struct {
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)

			// Skip unexported fields
			if !field.IsExported() {
				continue
			}

			schemaField := SchemaField{
				Name:     field.Name,
				Type:     field.Type.String(),
				Tag:      string(field.Tag),
				Required: !strings.Contains(field.Tag.Get("json"), "omitempty"),
			}

			// Recursively handle nested structs
			if field.Type.Kind() == reflect.Struct {
				nestedSchema := extractSchema(field.Type)
				schemaField.Fields = nestedSchema.Fields
			}

			schema.Fields = append(schema.Fields, schemaField)
		}
	}

	return schema
}
