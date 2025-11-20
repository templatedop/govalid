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

type eqValidator struct {
	pass       *codegen.Pass
	field      *ast.Field
	eqValue    string
	structName string
	ruleName   string
	parentPath string
}

var _ validator.Validator = (*eqValidator)(nil)

const eqKey = "%s-eq"

func (e *eqValidator) Validate() string {
	return fmt.Sprintf("!(t.%s == %s)", e.FieldName(), e.eqValue)
}

func (e *eqValidator) FieldName() string {
	return e.field.Names[0].Name
}
func (e *eqValidator) JSONFieldName() string {
	return validator.GetJSONTagName(e.field)
}

func (e *eqValidator) FieldPath() validator.FieldPath {
	return validator.NewFieldPath(e.structName, e.parentPath, e.FieldName())
}

func (e *eqValidator) Err() string {
	key := fmt.Sprintf(eqKey, e.structName+e.FieldPath().CleanedPath())

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
		// [@ERRVARIABLE] is the error returned when the field does not equal [@VALUE].
		[@ERRVARIABLE] = govaliderrors.ValidationError{Reason: "[@JSONFIELD] must equal [@VALUE]", Path: "[@PATH]", Type: "[@TYPE]"}
	`

	legacyErrVarName := fmt.Sprintf("Err%s%sEqValidation", e.structName, e.FieldName())
	currentErrVarName := e.ErrVariable()

	// Escape quotes in the value for error message
	escapedValue := strings.ReplaceAll(e.eqValue, `"`, `\"`)

	replacer := strings.NewReplacer(
		"[@ERRVARIABLE]", currentErrVarName,
		"[@LEGACYERRVAR]", legacyErrVarName,
		"[@JSONFIELD]", e.JSONFieldName(),
		"[@FIELD]", e.FieldName(),
		"[@PATH]", e.JSONFieldName(),
		"[@VALUE]", escapedValue,
		"[@TYPE]", e.ruleName,
	)

	if currentErrVarName != legacyErrVarName {
		return replacer.Replace(deprecationNoticeTemplate + errTemplate)
	}

	return replacer.Replace(errTemplate)
}

func (e *eqValidator) ErrVariable() string {
	return strings.ReplaceAll("Err[@PATH]EqValidation", "[@PATH]", e.FieldPath().CleanedPath())
}

func (e *eqValidator) Imports() []string {
	return []string{}
}

// ValidateEq creates a new eqValidator if the field is comparable and the eq marker is present.
func ValidateEq(input registry.ValidatorInput) validator.Validator {
	typ := input.Pass.TypesInfo.TypeOf(input.Field.Type)

	// Ensure the type is comparable (string, numeric, bool)
	if !types.Comparable(typ) {
		return nil
	}

	eqValue, ok := input.Expressions[markers.GoValidMarkerEq]
	if !ok {
		return nil
	}

	// For string types, wrap the value in quotes if not already quoted
	if basic, ok := typ.Underlying().(*types.Basic); ok && basic.Kind() == types.String {
		if !strings.HasPrefix(eqValue, `"`) && !strings.HasPrefix(eqValue, "`") {
			eqValue = fmt.Sprintf(`"%s"`, eqValue)
		}
	}

	return &eqValidator{
		pass:       input.Pass,
		field:      input.Field,
		eqValue:    eqValue,
		structName: input.StructName,
		ruleName:   input.RuleName,
		parentPath: input.ParentPath,
	}
}
