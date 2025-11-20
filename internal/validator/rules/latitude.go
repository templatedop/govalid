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

type latitudeValidator struct {
	pass       *codegen.Pass
	field      *ast.Field
	structName string
	ruleName   string
	parentPath string
}

var _ validator.Validator = (*latitudeValidator)(nil)

const latitudeKey = "%s-latitude"

func (v *latitudeValidator) Validate() string {
	fieldName := v.FieldName()
	return fmt.Sprintf("!validationhelper.IsValidLatitude(t.%s)", fieldName)
}

func (v *latitudeValidator) FieldName() string {
	return v.field.Names[0].Name
}
func (v *latitudeValidator) JSONFieldName() string {
	return validator.GetJSONTagName(v.field)
}

func (v *latitudeValidator) FieldPath() validator.FieldPath {
	return validator.NewFieldPath(v.structName, v.parentPath, v.FieldName())
}

func (v *latitudeValidator) Err() string {
	key := fmt.Sprintf(latitudeKey, v.structName+v.FieldPath().CleanedPath())

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
		// [@ERRVARIABLE] is the error returned when the field is not a valid latitude (-90 to 90).
		[@ERRVARIABLE] = govaliderrors.ValidationError{Reason: "[@JSONFIELD] must be a valid latitude (-90 to 90)", Path: "[@PATH]", Type: "[@TYPE]"}
	`

	legacyErrVarName := fmt.Sprintf("Err%s%sLatitudeValidation", v.structName, v.FieldName())
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

func (v *latitudeValidator) ErrVariable() string {
	return strings.ReplaceAll("Err[@PATH]LatitudeValidation", `[@PATH]`, v.FieldPath().CleanedPath())
}

func (v *latitudeValidator) Imports() []string {
	return []string{"github.com/templatedop/govalid/validation/validationhelper"}
}

// ValidateLatitude creates a new latitudeValidator for string types.
func ValidateLatitude(input registry.ValidatorInput) validator.Validator {
	typ := input.Pass.TypesInfo.TypeOf(input.Field.Type)

	// Check if it's a string type
	basic, ok := typ.Underlying().(*types.Basic)
	if !ok || basic.Kind() != types.String {
		return nil
	}

	return &latitudeValidator{
		pass:       input.Pass,
		field:      input.Field,
		structName: input.StructName,
		ruleName:   input.RuleName,
		parentPath: input.ParentPath,
	}
}
