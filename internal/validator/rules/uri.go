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

type uriValidator struct {
	pass       *codegen.Pass
	field      *ast.Field
	structName string
	ruleName   string
	parentPath string
}

var _ validator.Validator = (*uriValidator)(nil)

const uriKey = "%s-uri"

func (v *uriValidator) Validate() string {
	fieldName := v.FieldName()
	return fmt.Sprintf("!validationhelper.IsValidURI(t.%s)", fieldName)
}

func (v *uriValidator) FieldName() string {
	return v.field.Names[0].Name
}
func (v *uriValidator) JSONFieldName() string {
	return validator.GetJSONTagName(v.field)
}

func (v *uriValidator) FieldPath() validator.FieldPath {
	return validator.NewFieldPath(v.structName, v.parentPath, v.FieldName())
}

func (v *uriValidator) Err() string {
	key := fmt.Sprintf(uriKey, v.structName+v.FieldPath().CleanedPath())

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
		// [@ERRVARIABLE] is the error returned when the field is not a URI.
		[@ERRVARIABLE] = govaliderrors.ValidationError{Reason: "[@JSONFIELD] must be a URI", Path: "[@PATH]", Type: "[@TYPE]"}
	`

	legacyErrVarName := fmt.Sprintf("Err%s%sURIValidation", v.structName, v.FieldName())
	currentErrVarName := v.ErrVariable()

	replacer := strings.NewReplacer(
		"[@ERRVARIABLE]", currentErrVarName,
		"[@LEGACYERRVAR]", legacyErrVarName,
		"[@JSONFIELD]", v.JSONFieldName(),
		"[@FIELD]", v.FieldName(),
		"[@PATH]", v.JSONFieldName(),
		"[@TYPE]", v.ruleName,
	)

	if currentErrVarName != legacyErrVarName {
		return replacer.Replace(deprecationNoticeTemplate + errTemplate)
	}

	return replacer.Replace(errTemplate)
}

func (v *uriValidator) ErrVariable() string {
	return strings.ReplaceAll("Err[@PATH]URIValidation", `[@PATH]`, v.FieldPath().CleanedPath())
}

func (v *uriValidator) Imports() []string {
	return []string{"github.com/templatedop/govalid/validation/validationhelper"}
}

// ValidateURI creates a new uriValidator for string types.
func ValidateURI(input registry.ValidatorInput) validator.Validator {
	typ := input.Pass.TypesInfo.TypeOf(input.Field.Type)

	// Check if it's a string type
	basic, ok := typ.Underlying().(*types.Basic)
	if !ok || basic.Kind() != types.String {
		return nil
	}

	return &uriValidator{
		pass:       input.Pass,
		field:      input.Field,
		structName: input.StructName,
		ruleName:   input.RuleName,
		parentPath: input.ParentPath,
	}
}
