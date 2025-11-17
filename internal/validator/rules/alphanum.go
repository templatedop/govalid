// Package rules implements validation rules for fields in structs.
package rules

import (
	"fmt"
	"go/ast"
	"go/types"
	"strings"

	"github.com/gostaticanalysis/codegen"

	"github.com/templatedop/govalid/internal/validator"
	"github.com/templatedop/govalid/internal/validator/registry"
)

type alphanumValidator struct {
	pass       *codegen.Pass
	field      *ast.Field
	structName string
	ruleName   string
	parentPath string
}

var _ validator.Validator = (*alphanumValidator)(nil)

const alphanumKey = "%s-alphanum"

func (a *alphanumValidator) Validate() string {
	fieldName := a.FieldName()
	return fmt.Sprintf("!validationhelper.IsAlphanum(t.%s)", fieldName)
}

func (a *alphanumValidator) FieldName() string {
	return a.field.Names[0].Name
}

func (a *alphanumValidator) FieldPath() validator.FieldPath {
	return validator.NewFieldPath(a.structName, a.parentPath, a.FieldName())
}

func (a *alphanumValidator) Err() string {
	key := fmt.Sprintf(alphanumKey, a.FieldPath().CleanedPath())

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
		// [@ERRVARIABLE] is the error returned when the field contains non-alphanumeric characters.
		[@ERRVARIABLE] = govaliderrors.ValidationError{Reason: "field [@FIELD] must contain only alphanumeric characters", Path: "[@PATH]", Type: "[@TYPE]"}
	`

	legacyErrVarName := fmt.Sprintf("Err%s%sAlphanumValidation", a.structName, a.FieldName())
	currentErrVarName := a.ErrVariable()

	replacer := strings.NewReplacer(
		"[@ERRVARIABLE]", currentErrVarName,
		"[@LEGACYERRVAR]", legacyErrVarName,
		"[@FIELD]", a.FieldName(),
		"[@PATH]", a.FieldPath().String(),
		"[@TYPE]", a.ruleName,
	)

	if currentErrVarName != legacyErrVarName {
		return replacer.Replace(deprecationNoticeTemplate + errTemplate)
	}

	return replacer.Replace(errTemplate)
}

func (a *alphanumValidator) ErrVariable() string {
	return strings.ReplaceAll("Err[@PATH]AlphanumValidation", `[@PATH]`, a.FieldPath().CleanedPath())
}

func (a *alphanumValidator) Imports() []string {
	return []string{"github.com/templatedop/govalid/validation/validationhelper"}
}

// ValidateAlphanum creates a new alphanumValidator for string types.
func ValidateAlphanum(input registry.ValidatorInput) validator.Validator {
	typ := input.Pass.TypesInfo.TypeOf(input.Field.Type)

	// Check if it's a string type
	basic, ok := typ.Underlying().(*types.Basic)
	if !ok || basic.Kind() != types.String {
		return nil
	}

	return &alphanumValidator{
		pass:       input.Pass,
		field:      input.Field,
		structName: input.StructName,
		ruleName:   input.RuleName,
		parentPath: input.ParentPath,
	}
}
