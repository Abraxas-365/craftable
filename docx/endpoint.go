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

// Enhanced Header struct with type and default value
type Header struct {
	Name        string      `json:"name"`
	Value       string      `json:"value,omitempty"`
	Type        string      `json:"type,omitempty"`
	Description string      `json:"description,omitempty"`
	Required    bool        `json:"required"`
	Default     interface{} `json:"default,omitempty"`
}

// New PathParam struct for path parameters
type PathParam struct {
	Name        string      `json:"name"`
	Type        string      `json:"type"`
	Description string      `json:"description,omitempty"`
	Required    bool        `json:"required"`
	Default     interface{} `json:"default,omitempty"`
}

// Enhanced Endpoint struct with PathParams field
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

	// Parameters
	PathParams  []PathParam `json:"pathParams,omitempty"`
	QueryParams []Header    `json:"queryParams,omitempty"`

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

// Original WithHeader method
func (e *Endpoint) WithHeader(name, value string, required bool) *Endpoint {
	e.Headers = append(e.Headers, Header{
		Name:     name,
		Value:    value,
		Required: required,
	})
	return e
}

// Enhanced header method with more details
func (e *Endpoint) WithHeaderEx(name, value, paramType, description string, required bool) *Endpoint {
	e.Headers = append(e.Headers, Header{
		Name:        name,
		Value:       value,
		Type:        paramType,
		Description: description,
		Required:    required,
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

// Original query parameter method
func (e *Endpoint) WithQueryParam(name, description string, required bool) *Endpoint {
	e.QueryParams = append(e.QueryParams, Header{
		Name:        name,
		Description: description,
		Required:    required,
	})
	return e
}

// Enhanced query parameter method with type and default value
func (e *Endpoint) WithQueryParamEx(name, paramType, description string, required bool, defaultValue ...interface{}) *Endpoint {
	param := Header{
		Name:        name,
		Type:        paramType,
		Description: description,
		Required:    required,
	}

	// Set default value if provided
	if len(defaultValue) > 0 {
		param.Default = defaultValue[0]
	}

	e.QueryParams = append(e.QueryParams, param)
	return e
}

// New method for path parameters
func (e *Endpoint) WithPathParam(name, paramType, description string, required bool) *Endpoint {
	e.PathParams = append(e.PathParams, PathParam{
		Name:        name,
		Type:        paramType,
		Description: description,
		Required:    required,
	})
	return e
}

// Method for path parameters with default value
func (e *Endpoint) WithPathParamDefault(name, paramType, description string, required bool, defaultValue interface{}) *Endpoint {
	e.PathParams = append(e.PathParams, PathParam{
		Name:        name,
		Type:        paramType,
		Description: description,
		Required:    required,
		Default:     defaultValue,
	})
	return e
}

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
