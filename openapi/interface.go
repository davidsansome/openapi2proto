package openapi

import (
	"github.com/NYTimes/openapi2proto/internal/option"
)

const (
	optkeyDir = `dir`
)

// Option is used to pass options to several methods
type Option = option.Option

// resolver is used to resolve external references
type resolver struct{}
type resolveCtx struct {
	// this is used to qualify relative paths
	dir string

	// this holds the ready-to-be-inserted external references
	externalReferences map[string]interface{}

	// this holds the decoded content for each URL so we don't
	// have to keep fetching it
	cache map[string]interface{}
}

// GlobalOptions is used to store Protocol Buffers global options,
// such as package names
type GlobalOptions map[string]string

// Spec is the base struct for containing OpenAPI spec declarations.
type Spec struct {
	FileName string // internal use to pass file path
	OpenApi  string `yaml:"openapi" json:"openapi"`
	Info     struct {
		Title       string `yaml:"title" json:"title"`
		Description string `yaml:"description" json:"description"`
		Version     string `yaml:"version" json:"version"`
	} `yaml:"info" json:"info"`
	Servers       []*Server        `yaml:"servers" json:"servers"`
	Paths         map[string]*Path `yaml:"paths" json:"paths"`
	Components    Components       `yaml:"components" json:"components"`
	Extensions    []*Extension     `yaml:"x-extensions" json:"x-extensions"`
	GlobalOptions GlobalOptions    `yaml:"x-global-options" json:"x-global-options"`
}

type Components struct {
	Schemas    map[string]*Schema    `yaml:"schemas" json:"schemas"`
	Parameters map[string]*Parameter `yaml:"parameters" json:"parameters"`
}

type Server struct {
	Url string `yaml:"url" json:"url"`
}

// Extension is used to define Protocol Buffer extensions from
// within an OpenAPI spec. use `x-extentions` key.
type Extension struct {
	Base   string            `json:"base" yaml:"base"`
	Fields []*ExtensionField `json:"fields" yaml:"fields"`
}

// ExtensionField defines the fields to be added to the
// base message type
type ExtensionField struct {
	Name   string `yaml:"name" json:"name"`
	Type   string `yaml:"type" json:"type"`
	Number int    `yaml:"number" json:"number"`
}

// Path represents all of the endpoints and parameters available for a single
// path.
type Path struct {
	Get        *Endpoint  `yaml:"get" json:"get"`
	Put        *Endpoint  `yaml:"put" json:"put"`
	Post       *Endpoint  `yaml:"post" json:"post"`
	Patch      *Endpoint  `yaml:"patch" json:"patch"`
	Delete     *Endpoint  `yaml:"delete" json:"delete"`
	Parameters Parameters `yaml:"parameters" json:"parameters"`
}

// Parameter is a partial representation of OpenAPI parameter type
// (https://swagger.io/specification/#parameterObject)
type Parameter struct {
	Ref         string  `yaml:"$ref" json:"$ref"`
	Name        string  `yaml:"name" json:"name"`
	In          string  `yaml:"in,omitempty" json:"in,omitempty"`
	Description string  `yaml:"description" json:"description"`
	Required    bool    `yaml:"required,omitempty" json:"required,omitempty"`
	Schema      *Schema `yaml:"schema,omitempty" json:"schema,omitempty"`
	ProtoTag    int     `yaml:"x-proto-tag" json:"x-proto-tag"`
}

// Parameters is a slice of request parameters for a single endpoint.
type Parameters []*Parameter

// Response represents the response object in an OpenAPI spec.
type Response struct {
	Description string                      `yaml:"description" json:"description"`
	Content     map[string]*MediaTypeObject `yaml:"content" json:"content"`
}

// Endpoint represents an endpoint for a path in an OpenAPI spec.
type Endpoint struct {
	Path          string                 `yaml:"-" json:"-"` // this is added internally
	Verb          string                 `yaml:"-" json:"-"` // this is added internally
	Summary       string                 `yaml:"summary" json:"summary"`
	Description   string                 `yaml:"description" json:"description"`
	Parameters    Parameters             `yaml:"parameters" json:"parameters"`
	RequestBody   *RequestBody           `yaml:"requestBody" json:"requestBody"`
	Tags          []string               `yaml:"tags" json:"tags"`
	Responses     map[string]*Response   `yaml:"responses" json:"responses"`
	OperationID   string                 `yaml:"operationId" json:"operationId"`
	CustomOptions map[string]interface{} `yaml:"x-options" json:"x-options"`
}

type RequestBody struct {
	Ref string `yaml:"$ref" json:"$ref"`

	Description string                      `yaml:"description" json:"description"`
	Required    bool                        `yaml:"required" json:"required"`
	Content     map[string]*MediaTypeObject `yaml:"content" json:"content"`
}

type MediaTypeObject struct {
	Schema *Schema `yaml:"schema" json:"schema"`
}

// Model represents a model definition from an OpenAPI spec.
type Model struct {
	Properties map[string]*Schema `yaml:"properties" json:"properties"`
	Name       string
	Depth      int
}

// Schema represent Model properties in an OpenAPI spec.
type Schema struct {
	isNil bool

	// if this schema refers to a definition found elsewhere, this value
	// is used. Note that if present, this takes precedence over other values
	Ref string `yaml:"$ref" json:"$ref"`

	Description string `yaml:"description,omitempty" json:"description,omitempty"`
	// scalar
	// https://github.com/OAI/OpenAPI-Specification/blob/master/versions/3.0.0.md#schemaObject
	Type   string   `yaml:"type" json:"type"`
	Format string   `yaml:"format,omitempty" json:"format,omitempty"`
	Enum   []string `yaml:"enum,omitempty" json:"enum,omitempty"`

	ProtoName string `yaml:"-" json:"-"`
	ProtoTag  int    `yaml:"x-proto-tag" json:"x-proto-tag"`

	// objects
	Required             []string           `yaml:"required" json:"required"`
	Properties           map[string]*Schema `yaml:"properties" json:"properties"`
	AdditionalProperties *Schema            `yaml:"additionalProperties" json:"additionalProperties"`

	// is an array
	Items *Schema `yaml:"items" json:"items"`

	AnyOf []*Schema `yaml:"anyOf" json:"anyOf"`

	// validation (regex pattern, max/min length)
	Pattern   string `yaml:"pattern,omitempty" json:"pattern,omitempty"`
	MaxLength int    `yaml:"maxLength,omitempty" json:"maxLength,omitempty"`
	MinLength int    `yaml:"minLength,omitempty" json:"minLength,omitempty"`
	Maximum   int    `yaml:"maximum,omitempty" json:"maximum,omitempty"`
	Minimum   int    `yaml:"minimum,omitempty" json:"minimum,omitempty"`
}
