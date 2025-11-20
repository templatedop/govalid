// Package validator implements rules for validating fields.
package validator

import (
	"go/ast"
	"reflect"
	"strings"
)

// Validator is an interface for validating fields in structs.
type Validator interface {
	Validate() string
	FieldName() string
	JSONFieldName() string
	FieldPath() FieldPath
	Err() string
	ErrVariable() string
	Imports() []string
}

// GeneratorMemory is a map used to track the state of generated validators.
var GeneratorMemory = map[string]bool{}

// GetJSONTagName extracts the JSON field name from a struct field's tag.
// If no json tag exists, it returns the field name.
func GetJSONTagName(field *ast.Field) string {
	if field.Tag == nil {
		return field.Names[0].Name
	}

	tagValue := strings.Trim(field.Tag.Value, "`")
	tag := reflect.StructTag(tagValue)

	jsonTag := tag.Get("json")
	if jsonTag == "" {
		return field.Names[0].Name
	}

	// Handle json tag options like "name,omitempty"
	parts := strings.Split(jsonTag, ",")
	if len(parts) > 0 && parts[0] != "" && parts[0] != "-" {
		return parts[0]
	}

	return field.Names[0].Name
}
