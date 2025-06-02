package validatex

import (
	"errors"
	"reflect"

	"github.com/Abraxas-365/craftable/errx"
)

// Validatable is an interface for types that provide their own validation
type Validatable interface {
	Validate() error
}

// Validate validates a struct based on the validatex tags
func Validate(obj any) error {
	// Check if obj implements Validatable
	if validatable, ok := obj.(Validatable); ok {
		return validatable.Validate()
	}

	// Get fields to validate
	fields, err := structFields(obj)
	if err != nil {
		if err == ErrNotStruct {
			return ValidatorErrors.New(ErrInvalidStruct)
		}
		return err
	}

	// Validate each field
	var validationErrors ValidationErrors

	for fieldName, field := range fields {
		for _, rule := range field.Rules {
			// Get validation function
			fn, ok := getValidationFunc(rule.Name)
			if !ok {
				validationErrors = append(validationErrors, NewValidationError(
					fieldName, rule.Name, rule.Param, field.Value,
					"unknown validation rule",
				))
				continue
			}

			// Skip required check if field is zero and rule is not required
			if rule.Name != "required" && isZero(field.Value) {
				continue
			}

			// Apply the validation rule
			if !fn(field.Value, rule.Param) {
				validationErrors = append(validationErrors, NewValidationError(
					fieldName, rule.Name, rule.Param, field.Value, "",
				))
			}
		}
	}

	if len(validationErrors) > 0 {
		return validationErrors
	}

	return nil
}

// ValidateWithErrx validates a struct and returns an errx.Error if validation fails
func ValidateWithErrx(obj any) *errx.Error {
	err := Validate(obj)
	if err == nil {
		return nil
	}

	// Check if we have validation errors
	if validationErrors, ok := err.(ValidationErrors); ok {
		return validationErrors.ToErrx()
	}

	// Handle other error types
	var validatorErr *errx.Error
	if errors.As(err, &validatorErr) {
		return validatorErr
	}

	// Default to a generic validation error
	return ValidatorErrors.NewWithMessage(ErrValidationFailed, err.Error())
}

// ValidateField validates a single value against a validation rule
func ValidateField(value any, rule string) error {
	// Parse the rule
	rules := parseTag(rule)
	if len(rules) == 0 {
		return nil
	}

	// Apply each validation rule
	var validationErrors ValidationErrors

	for _, r := range rules {
		// Get validation function
		fn, ok := getValidationFunc(r.Name)
		if !ok {
			validationErrors = append(validationErrors, NewValidationError(
				"", r.Name, r.Param, value,
				"unknown validation rule",
			))
			continue
		}

		// Skip required check if field is zero and rule is not required
		if r.Name != "required" && isZero(value) {
			continue
		}

		// Apply the validation rule
		if !fn(value, r.Param) {
			validationErrors = append(validationErrors, NewValidationError(
				"", r.Name, r.Param, value, "",
			))
		}
	}

	if len(validationErrors) > 0 {
		return validationErrors
	}

	return nil
}

// ValidateFieldWithErrx validates a field and returns an errx.Error if validation fails
func ValidateFieldWithErrx(fieldName string, value any, rule string) *errx.Error {
	err := ValidateField(value, rule)
	if err == nil {
		return nil
	}

	// Check if we have validation errors
	if validationErrors, ok := err.(ValidationErrors); ok {
		// Add the field name to each error
		for i := range validationErrors {
			validationErrors[i].Field = fieldName
		}
		return validationErrors.ToErrx()
	}

	// Handle other error types
	return ValidatorErrors.NewWithMessage(
		ErrValidationFailed,
		"Validation failed for field "+fieldName+": "+err.Error(),
	).WithDetail("field", fieldName)
}

// MustValidate validates a struct and panics if validation fails
func MustValidate(obj any) {
	if err := Validate(obj); err != nil {
		panic(err)
	}
}

// CustomValidator allows customization of validation behavior
type CustomValidator struct {
	TagName string
	Rules   map[string]ValidationFunc
}

// NewValidator creates a new custom validator
func NewValidator() *CustomValidator {
	return &CustomValidator{
		TagName: "validatex",
		Rules:   make(map[string]ValidationFunc),
	}
}

// RegisterRule registers a custom validation rule
func (v *CustomValidator) RegisterRule(name string, fn ValidationFunc) *CustomValidator {
	v.Rules[name] = fn
	return v
}

// WithTagName sets the tag name to use for validation
func (v *CustomValidator) WithTagName(tagName string) *CustomValidator {
	v.TagName = tagName
	return v
}

// Validate validates a struct using this validator's configuration
func (v *CustomValidator) Validate(obj any) error {
	// Use a modified version of the standard Validate function
	// that uses this validator's tag name and rules

	// Check if obj implements Validatable
	if validatable, ok := obj.(Validatable); ok {
		return validatable.Validate()
	}

	val := reflect.ValueOf(obj)

	// If obj is a pointer, dereference it
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return nil
		}
		val = val.Elem()
	}

	// Only process structs
	if val.Kind() != reflect.Struct {
		return ValidatorErrors.New(ErrInvalidStruct)
	}

	typ := val.Type()
	var validationErrors ValidationErrors

	// Process all fields in the struct
	for i := range typ.NumField() {
		field := typ.Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		// Get the validatex tag
		tag := field.Tag.Get(v.TagName)
		if tag == "" || tag == "-" {
			continue
		}

		// Get the field value
		fieldValue := val.Field(i)
		fieldInterface := fieldValue.Interface()

		// Parse the tag
		rules := parseTag(tag)

		for _, rule := range rules {
			// Try custom rules first
			var fn ValidationFunc
			var ok bool

			fn, ok = v.Rules[rule.Name]
			if !ok {
				// Then try built-in rules
				fn, ok = getValidationFunc(rule.Name)
			}

			if !ok {
				validationErrors = append(validationErrors, NewValidationError(
					field.Name, rule.Name, rule.Param, fieldInterface,
					"unknown validation rule",
				))
				continue
			}

			// Skip required check if field is zero and rule is not required
			if rule.Name != "required" && isZero(fieldInterface) {
				continue
			}

			// Apply the validation rule
			if !fn(fieldInterface, rule.Param) {
				validationErrors = append(validationErrors, NewValidationError(
					field.Name, rule.Name, rule.Param, fieldInterface, "",
				))
			}
		}

		// For struct recursion, dereference pointers
		actualFieldValue := fieldValue
		if fieldValue.Kind() == reflect.Ptr && !fieldValue.IsNil() {
			actualFieldValue = fieldValue.Elem()
		}

		// If the field is a struct, recursively validate it
		if actualFieldValue.Kind() == reflect.Struct {
			// Skip if already validatable
			if _, ok := actualFieldValue.Interface().(Validatable); !ok {
				if err := v.Validate(actualFieldValue.Interface()); err != nil {
					if validationErrs, ok := err.(ValidationErrors); ok {
						// Add field name prefix to nested errors
						for _, validationErr := range validationErrs {
							validationErr.Field = field.Name + "." + validationErr.Field
							validationErrors = append(validationErrors, validationErr)
						}
					} else {
						// Add generic error
						validationErrors = append(validationErrors, NewValidationError(
							field.Name, "", "", fieldInterface, err.Error(),
						))
					}
				}
			}
		}
	}

	if len(validationErrors) > 0 {
		return validationErrors
	}

	return nil
}

// ValidateWithErrx validates a struct using this validator and returns an errx.Error
func (v *CustomValidator) ValidateWithErrx(obj any) *errx.Error {
	err := v.Validate(obj)
	if err == nil {
		return nil
	}

	// Check if we have validation errors
	if validationErrors, ok := err.(ValidationErrors); ok {
		return validationErrors.ToErrx()
	}

	// Handle other error types
	var validatorErr *errx.Error
	if errors.As(err, &validatorErr) {
		return validatorErr
	}

	// Default to a generic validation error
	return ValidatorErrors.NewWithMessage(ErrValidationFailed, err.Error())
}
