package docx

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

type CurlGenerator struct {
	BaseURL string
}

func NewCurlGenerator(baseURL string) *CurlGenerator {
	return &CurlGenerator{
		BaseURL: strings.TrimRight(baseURL, "/"),
	}
}

func (g *CurlGenerator) GenerateCurl(endpoint *Endpoint) (string, error) {
	var buf bytes.Buffer

	// Start with the base curl command
	buf.WriteString("curl")

	// Add the HTTP method
	buf.WriteString(fmt.Sprintf(" -X %s", endpoint.Method))

	// Build URL with query parameters
	fullPath := g.BaseURL + endpoint.Path

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
		fullPath += "?" + strings.Join(queryParams, "&")
	}

	buf.WriteString(fmt.Sprintf(" \"%s\"", fullPath))

	// Add headers
	for _, header := range endpoint.Headers {
		if header.Value != "" {
			buf.WriteString(fmt.Sprintf(" -H \"%s: %s\"", header.Name, header.Value))
		} else {
			buf.WriteString(fmt.Sprintf(" -H \"%s: <VALUE>\"", header.Name))
		}
	}

	// Add authentication
	switch endpoint.Auth {
	case Basic:
		if details, ok := endpoint.AuthDetails["username"]; ok {
			username := details
			password := endpoint.AuthDetails["password"]
			buf.WriteString(fmt.Sprintf(" -u \"%s:%s\"", username, password))
		} else {
			buf.WriteString(" -u \"<USERNAME>:<PASSWORD>\"")
		}
	case Bearer:
		if token, ok := endpoint.AuthDetails["token"]; ok {
			buf.WriteString(fmt.Sprintf(" -H \"Authorization: Bearer %s\"", token))
		} else {
			buf.WriteString(" -H \"Authorization: Bearer <TOKEN>\"")
		}
	case ApiKey:
		if name, ok := endpoint.AuthDetails["name"]; ok {
			value := endpoint.AuthDetails["value"]
			in := endpoint.AuthDetails["in"]
			if in == "header" {
				buf.WriteString(fmt.Sprintf(" -H \"%s: %s\"", name, value))
			}
		}
	}

	// Add request body if available
	if endpoint.RequestExample != nil {
		bodyJSON, err := json.MarshalIndent(endpoint.RequestExample, "", "  ")
		if err != nil {
			return "", err
		}
		buf.WriteString(fmt.Sprintf(" -d '%s'", string(bodyJSON)))
	} else if endpoint.RequestBody != nil {
		buf.WriteString(" -d '<REQUEST_BODY>'")
	}

	return buf.String(), nil
}

func (g *CurlGenerator) GenerateAllCurls(router *RouterDoc) (map[string]string, error) {
	results := make(map[string]string)

	for _, endpoint := range router.Endpoints {
		curl, err := g.GenerateCurl(endpoint)
		if err != nil {
			return nil, err
		}

		key := string(endpoint.Method) + " " + endpoint.Path
		results[key] = curl
	}

	return results, nil
}
