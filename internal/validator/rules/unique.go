// Package rules implements validation rules for fields in structs.
package rules

import (
	"fmt"
	"go/ast"
	"go/types"
	"strings"

	"github.com/gostaticanalysis/codegen"

	"github.com/sivchari/govalid/internal/validator"
	"github.com/sivchari/govalid/internal/validator/registry"
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
		[@ERRVARIABLE] = govaliderrors.ValidationError{Reason:"field [@FIELD] must contain unique values",Path:"[@PATH]",Type:"[@TYPE]"}
	`

	legacyErrVarName := fmt.Sprintf("Err%s%sUniqueValidation", u.structName, u.FieldName())
	currentErrVarName := u.ErrVariable()

	replacer := strings.NewReplacer(
		"[@ERRVARIABLE]", currentErrVarName,
		"[@LEGACYERRVAR]", legacyErrVarName,
		"[@FIELD]", u.FieldName(),
		"[@PATH]", u.FieldPath().String(),
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
	typ := input.Pass.TypesInfo.TypeOf(input.Field.Type)

	// Check if it's a slice or array
	var elemType types.Type
	switch t := typ.Underlying().(type) {
	case *types.Slice:
		elemType = t.Elem()
	case *types.Array:
		elemType = t.Elem()
	default:
		return nil
	}

	// Element type must be comparable
	if !types.Comparable(elemType) {
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
