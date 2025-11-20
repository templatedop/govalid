// Package rules implements validation rules for fields in structs.
package rules

import (
	"fmt"
	"go/ast"
	"strings"

	"github.com/gostaticanalysis/codegen"

	"github.com/templatedop/govalid/internal/validator"
	"github.com/templatedop/govalid/internal/validator/registry"
)

type uniqueValidator struct {
	pass       *codegen.Pass
	field      *ast.Field
	structName string
	ruleName   string
	parentPath string
}

var _ validator.Validator = (*uniqueValidator)(nil)

const uniqueKey = "%s-unique"

func (u *uniqueValidator) Validate() string {
	fieldName := u.FieldName()
	// Generate inline uniqueness check using a map
	return fmt.Sprintf(`func() bool {
		seen := make(map[interface{}]struct{})
		for _, v := range t.%s {
			if _, exists := seen[v]; exists {
				return true
			}
			seen[v] = struct{}{}
		}
		return false
	}()`, fieldName)
}

func (u *uniqueValidator) FieldName() string {
	return u.field.Names[0].Name
}
func (u *uniqueValidator) JSONFieldName() string {
	return validator.GetJSONTagName(u.field)
}

func (u *uniqueValidator) FieldPath() validator.FieldPath {
	return validator.NewFieldPath(u.structName, u.parentPath, u.FieldName())
}

func (u *uniqueValidator) Err() string {
	key := fmt.Sprintf(uniqueKey, u.structName+u.FieldPath().CleanedPath())

	if validator.GeneratorMemory[key] {
		return ""
	}

	validator.GeneratorMemory[key] = true

	const deprecationNoticeTemplate = `
		// Deprecated: Use [@ERRVARIABLE]
		//
		// [@LEGACYERRVAR] is deprecated and is kept for compatibility purpose.
		[@LEGACYERRVAR] = [@ERRVARIABLE]
	`

	const errTemplate = `
		// [@ERRVARIABLE] is the error returned when the field contains duplicate values.
		[@ERRVARIABLE] = govaliderrors.ValidationError{Reason: "[@JSONFIELD] must contain unique values", Path: "[@PATH]", Type: "[@TYPE]"}
	`

	legacyErrVarName := fmt.Sprintf("Err%s%sUniqueValidation", u.structName, u.FieldName())
	currentErrVarName := u.ErrVariable()

	replacer := strings.NewReplacer(
		"[@ERRVARIABLE]", currentErrVarName,
		"[@LEGACYERRVAR]", legacyErrVarName,
		"[@JSONFIELD]", u.JSONFieldName(),
		"[@FIELD]", u.FieldName(),
		"[@PATH]", u.JSONFieldName(),
		"[@TYPE]", u.ruleName,
	)

	if currentErrVarName != legacyErrVarName {
		return replacer.Replace(deprecationNoticeTemplate + errTemplate)
	}

	return replacer.Replace(errTemplate)
}

func (u *uniqueValidator) ErrVariable() string {
	return strings.ReplaceAll("Err[@PATH]UniqueValidation", "[@PATH]", u.FieldPath().CleanedPath())
}

func (u *uniqueValidator) Imports() []string {
	return []string{}
}

// ValidateUnique creates a new uniqueValidator for slice/array types with comparable elements.
func ValidateUnique(input registry.ValidatorInput) validator.Validator {
	// For slices and arrays, we always allow the unique validator
	// Type checking is done at compile time when the generated code is built

	// Simple AST-based check for slice or array types
	fieldType := input.Field.Type

	// Check if it's a slice (ast.ArrayType with nil Len) or array (ast.ArrayType with Len)
	switch t := fieldType.(type) {
	case *ast.ArrayType:
		// Both slices and arrays are represented as ArrayType in AST
		// Slices have Len == nil, arrays have Len != nil
		// We support both for unique validation
		_ = t // Valid for unique
	default:
		// Not a slice or array, skip
		return nil
	}

	return &uniqueValidator{
		pass:       input.Pass,
		field:      input.Field,
		structName: input.StructName,
		ruleName:   input.RuleName,
		parentPath: input.ParentPath,
	}
}
