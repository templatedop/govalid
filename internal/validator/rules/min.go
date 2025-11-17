// Package rules implements validation rules for fields in structs.
package rules

import (
	"fmt"
	"go/ast"
	"go/types"
	"strings"

	"github.com/gostaticanalysis/codegen"

	"github.com/templatedop/govalid/internal/markers"
	"github.com/templatedop/govalid/internal/validator"
	"github.com/templatedop/govalid/internal/validator/registry"
)

type minValidator struct {
	pass       *codegen.Pass
	field      *ast.Field
	minValue   string
	structName string
	ruleName   string
	parentPath string
}

var _ validator.Validator = (*minValidator)(nil)

const minKey = "%s-min"

func (m *minValidator) Validate() string {
	return fmt.Sprintf("!(t.%s >= %s)", m.FieldName(), m.minValue)
}

func (m *minValidator) FieldName() string {
	return m.field.Names[0].Name
}

func (m *minValidator) FieldPath() validator.FieldPath {
	return validator.NewFieldPath(m.structName, m.parentPath, m.FieldName())
}

func (m *minValidator) Err() string {
	key := fmt.Sprintf(minKey, m.structName+m.FieldPath().CleanedPath())

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
		// [@ERRVARIABLE] is the error returned when the value of the field is less than the minimum of [@VALUE].
		[@ERRVARIABLE] = govaliderrors.ValidationError{Reason: "field [@FIELD] must be greater than or equal to [@VALUE]", Path: "[@PATH]", Type: "[@TYPE]"}
	`

	legacyErrVarName := fmt.Sprintf("Err%s%sMinValidation", m.structName, m.FieldName())
	currentErrVarName := m.ErrVariable()

	replacer := strings.NewReplacer(
		"[@ERRVARIABLE]", currentErrVarName,
		"[@LEGACYERRVAR]", legacyErrVarName,
		"[@FIELD]", m.FieldName(),
		"[@PATH]", m.FieldPath().String(),
		"[@VALUE]", m.minValue,
		"[@TYPE]", m.ruleName,
	)

	if currentErrVarName != legacyErrVarName {
		return replacer.Replace(deprecationNoticeTemplate + errTemplate)
	}

	return replacer.Replace(errTemplate)
}

func (m *minValidator) ErrVariable() string {
	return strings.ReplaceAll("Err[@PATH]MinValidation", "[@PATH]", m.FieldPath().CleanedPath())
}

func (m *minValidator) Imports() []string {
	return []string{}
}

// ValidateMin creates a new minValidator if the field type is numeric and the min marker is present.
func ValidateMin(input registry.ValidatorInput) validator.Validator {
	typ := input.Pass.TypesInfo.TypeOf(input.Field.Type)
	basic, ok := typ.Underlying().(*types.Basic)

	if !ok || (basic.Info()&types.IsNumeric) == 0 {
		return nil
	}

	minValue, ok := input.Expressions[markers.GoValidMarkerMin]
	if !ok {
		return nil
	}

	return &minValidator{
		pass:       input.Pass,
		field:      input.Field,
		minValue:   minValue,
		structName: input.StructName,
		ruleName:   input.RuleName,
		parentPath: input.ParentPath,
	}
}
