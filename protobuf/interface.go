package protobuf

import "io"

const (
	priorityEnum = iota
	priorityMessage
	priorityExtension
	priorityService
)

var (
	BoolType   = newBuiltin("bool")
	BytesType  = newBuiltin("bytes")
	DoubleType = newBuiltin("double")
	FloatType  = newBuiltin("float")
	Int32Type  = newBuiltin("int32")
	Int64Type  = newBuiltin("int64")
	StringType = newBuiltin("string")

	// Boxed types
	AnyType         = NewMessage("google.protobuf.Any")
	BoolValueType   = NewMessage("google.protobuf.BoolValue")
	BytesValueType  = NewMessage("google.protobuf.BytesValue")
	DoubleValueType = NewMessage("google.protobuf.DoubleValue")
	FloatValueType  = NewMessage("google.protobuf.FloatValue")
	Int32ValueType  = NewMessage("google.protobuf.Int32Value")
	Int64ValueType  = NewMessage("google.protobuf.Int64Value")
	NullValueType   = NewMessage("google.protobuf.NullValue")
	StringValueType   = NewMessage("google.protobuf.StringValue")
)

var (
	emptyMessage = NewMessage("google.protobuf.Empty")
)

type Encoder struct {
	dst    io.Writer
	indent string
}

type GlobalOption struct {
	name string
	value string
}

// A protocol buffers definition is in itself one big message type,
// but with extra options.
type Package struct {
	name     string
	imports  []string
	children []Type
	options  []*GlobalOption
}

type Type interface {
	Name() string
	Priority() int
}

type Container interface {
	Type
	AddType(Type)
	Children() []Type
}

type Enum struct {
	comment  string
	elements []interface{}
	name     string
}

type Map struct {
	key   Type
	value Type
}

// Builtin
type Builtin string

// Message is a composite type
type Message struct {
	children []Type
	comment  string
	fields   []*Field
	name     string
}

type Field struct {
	comment  string
	index    int
	name     string
	repeated bool
	typ      Type
}

type ExtensionField struct {
	name   string
	typ    string
	number int
}

type Extension struct {
	base   string
	fields []*ExtensionField
}

// RPC represents an RPC call associated with a Service
type RPC struct {
	comment   string
	name      string
	parameter Type
	response  Type

	options []interface{}
}

// Service defines a service with many RPC endpoints
type Service struct {
	name string
	rpcs []*RPC
}

type HTTPAnnotation struct {
	method string
	path   string
	body   string
}

type RPCOption struct {
	name  string
	value interface{}
}

// Reference is a special type of Type that can pass the
// protobuf.Type system, but requires that  it be resolved
// at runtime to get the actual Type behind it. This is
// used to resolve circular dependencies that are found
// during compilation phase
type Reference struct {
	name     string
	resolver func(string) (Type, error)
}
