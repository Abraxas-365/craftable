package docx

import (
	"reflect"
)

type HTTPMethod string

const (
	GET     HTTPMethod = "GET"
	POST    HTTPMethod = "POST"
	PUT     HTTPMethod = "PUT"
	DELETE  HTTPMethod = "DELETE"
	PATCH   HTTPMethod = "PATCH"
	OPTIONS HTTPMethod = "OPTIONS"
	HEAD    HTTPMethod = "HEAD"
)

type Authentication string

const (
	None       Authentication = "none"
	Basic      Authentication = "basic"
	Bearer     Authentication = "bearer"
	ApiKey     Authentication = "apiKey"
	OAuth2     Authentication = "oauth2"
	CustomAuth Authentication = "custom"
)

type Header struct {
	Name        string `json:"name"`
	Value       string `json:"value,omitempty"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required"`
}

type Endpoint struct {
	Path        string     `json:"path"`
	Method      HTTPMethod `json:"method"`
	Description string     `json:"description"`
	Summary     string     `json:"summary,omitempty"`
	Tags        []string   `json:"tags,omitempty"`

	// DTO mappings
	RequestBody    interface{} `json:"-"`
	RequestSchema  interface{} `json:"requestSchema,omitempty"`
	ResponseBody   interface{} `json:"-"`
	ResponseSchema interface{} `json:"responseSchema,omitempty"`

	// Authentication and headers
	Auth        Authentication    `json:"auth"`
	AuthDetails map[string]string `json:"authDetails,omitempty"`
	Headers     []Header          `json:"headers,omitempty"`
	Cookies     []Header          `json:"cookies,omitempty"`

	// Query parameters
	QueryParams []Header `json:"queryParams,omitempty"`

	// Example values for documentation
	RequestExample  interface{} `json:"requestExample,omitempty"`
	ResponseExample interface{} `json:"responseExample,omitempty"`
}

func NewEndpoint(path string, method HTTPMethod) *Endpoint {
	return &Endpoint{
		Path:   path,
		Method: method,
		Auth:   None,
		Tags:   []string{},
	}
}

func (e *Endpoint) WithDescription(desc string) *Endpoint {
	e.Description = desc
	return e
}

func (e *Endpoint) WithSummary(summary string) *Endpoint {
	e.Summary = summary
	return e
}

func (e *Endpoint) WithTags(tags ...string) *Endpoint {
	e.Tags = append(e.Tags, tags...)
	return e
}

func (e *Endpoint) WithAuth(auth Authentication, details map[string]string) *Endpoint {
	e.Auth = auth
	e.AuthDetails = details
	return e
}

// Fixed WithHeader method to use slice instead of map
func (e *Endpoint) WithHeader(name, value string, required bool) *Endpoint {
	e.Headers = append(e.Headers, Header{
		Name:     name,
		Value:    value,
		Required: required,
	})
	return e
}

func (e *Endpoint) WithCookie(name, description string, required bool) *Endpoint {
	e.Cookies = append(e.Cookies, Header{
		Name:        name,
		Description: description,
		Required:    required,
	})
	return e
}

func (e *Endpoint) WithQueryParam(name, description string, required bool) *Endpoint {
	e.QueryParams = append(e.QueryParams, Header{
		Name:        name,
		Description: description,
		Required:    required,
	})
	return e
}

// Update the WithRequestDTO and WithResponseDTO methods
func (e *Endpoint) WithRequestDTO(dto interface{}) *Endpoint {
	e.RequestBody = dto
	e.RequestSchema = extractSchema(reflect.TypeOf(dto))
	return e
}

func (e *Endpoint) WithResponseDTO(dto interface{}) *Endpoint {
	e.ResponseBody = dto
	e.ResponseSchema = extractSchema(reflect.TypeOf(dto))
	return e
}

func (e *Endpoint) WithRequestExample(example interface{}) *Endpoint {
	e.RequestExample = example
	return e
}

func (e *Endpoint) WithResponseExample(example interface{}) *Endpoint {
	e.ResponseExample = example
	return e
}
