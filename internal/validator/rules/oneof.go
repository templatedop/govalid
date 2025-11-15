// Package rules implements validation rules for fields in structs.
package rules

import (
	"fmt"
	"go/ast"
	"go/types"
	"strings"

	"github.com/gostaticanalysis/codegen"

	"github.com/sivchari/govalid/internal/markers"
	"github.com/sivchari/govalid/internal/validator"
	"github.com/sivchari/govalid/internal/validator/registry"
)

type oneofValidator struct {
	pass       *codegen.Pass
	field      *ast.Field
	values     string
	structName string
	ruleName   string
	parentPath string
}

var _ validator.Validator = (*oneofValidator)(nil)

const oneofKey = "%s-oneof"

func (o *oneofValidator) Validate() string {
	fieldName := o.FieldName()
	// Generate a validation that checks if the value is in the list
	values := strings.Split(o.values, " ")
	var conditions []string
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		// Wrap in quotes if not already quoted
		if !strings.HasPrefix(v, `"`) && !strings.HasPrefix(v, "`") {
			v = fmt.Sprintf(`"%s"`, v)
		}
		conditions = append(conditions, fmt.Sprintf("t.%s == %s", fieldName, v))
	}
	if len(conditions) == 0 {
		return "false"
	}
	return fmt.Sprintf("!(%s)", strings.Join(conditions, " || "))
}

func (o *oneofValidator) FieldName() string {
	return o.field.Names[0].Name
}

func (o *oneofValidator) FieldPath() validator.FieldPath {
	return validator.NewFieldPath(o.structName, o.parentPath, o.FieldName())
}

func (o *oneofValidator) Err() string {
	key := fmt.Sprintf(oneofKey, o.structName+o.FieldPath().CleanedPath())

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
		// [@ERRVARIABLE] is the error returned when the field is not one of the allowed values.
		[@ERRVARIABLE] = govaliderrors.ValidationError{Reason: "field [@FIELD] must be one of [@VALUES]", Path: "[@PATH]", Type: "[@TYPE]"}
	`

	legacyErrVarName := fmt.Sprintf("Err%s%sOneofValidation", o.structName, o.FieldName())
	currentErrVarName := o.ErrVariable()

	replacer := strings.NewReplacer(
		"[@ERRVARIABLE]", currentErrVarName,
		"[@LEGACYERRVAR]", legacyErrVarName,
		"[@FIELD]", o.FieldName(),
		"[@PATH]", o.FieldPath().String(),
		"[@VALUES]", o.values,
		"[@TYPE]", o.ruleName,
	)

	if currentErrVarName != legacyErrVarName {
		return replacer.Replace(deprecationNoticeTemplate + errTemplate)
	}

	return replacer.Replace(errTemplate)
}

func (o *oneofValidator) ErrVariable() string {
	return strings.ReplaceAll("Err[@PATH]OneofValidation", "[@PATH]", o.FieldPath().CleanedPath())
}

func (o *oneofValidator) Imports() []string {
	return []string{}
}

// ValidateOneof creates a new oneofValidator for string/numeric types.
func ValidateOneof(input registry.ValidatorInput) validator.Validator {
	typ := input.Pass.TypesInfo.TypeOf(input.Field.Type)

	// Check if it's a string or numeric type
	basic, ok := typ.Underlying().(*types.Basic)
	if !ok || (basic.Kind() != types.String && (basic.Info()&types.IsNumeric) == 0) {
		return nil
	}

	values, ok := input.Expressions[markers.GoValidMarkerOneof]
	if !ok {
		return nil
	}

	return &oneofValidator{
		pass:       input.Pass,
		field:      input.Field,
		values:     values,
		structName: input.StructName,
		ruleName:   input.RuleName,
		parentPath: input.ParentPath,
	}
}
