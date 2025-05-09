package dtox

import (
	"errors"
	"fmt"
	"reflect"
	"sync"

	"github.com/Abraxas-365/craftale/errx"
)

// Mapper provides type-safe conversion between DTOs and domain models
type Mapper[TDto any, TModel any] struct {
	dtoToModelFn  func(dto TDto) (TModel, error)
	modelToDtoFn  func(model TModel) (TDto, error)
	fieldMappings map[string]string
	ignoreFields  map[string]bool
	validationFn  func(dto TDto) error
	strictMode    bool
	options       *Options
	fieldCache    *sync.Map
}

// NewMapper creates a new mapper for converting between DTO and model types
func NewMapper[TDto any, TModel any]() *Mapper[TDto, TModel] {
	return &Mapper[TDto, TModel]{
		fieldMappings: make(map[string]string),
		ignoreFields:  make(map[string]bool),
		options:       defaultOptions(),
		fieldCache:    &sync.Map{},
	}
}

// WithCustomDtoToModel sets a custom function for converting from DTO to model
func (m *Mapper[TDto, TModel]) WithCustomDtoToModel(fn func(dto TDto) (TModel, error)) *Mapper[TDto, TModel] {
	m.dtoToModelFn = fn
	return m
}

// WithCustomModelToDto sets a custom function for converting from model to DTO
func (m *Mapper[TDto, TModel]) WithCustomModelToDto(fn func(model TModel) (TDto, error)) *Mapper[TDto, TModel] {
	m.modelToDtoFn = fn
	return m
}

// WithFieldMapping adds a field name mapping from DTO field to model field
func (m *Mapper[TDto, TModel]) WithFieldMapping(dtoField, modelField string) *Mapper[TDto, TModel] {
	m.fieldMappings[dtoField] = modelField
	return m
}

// WithIgnoreField specifies a field to ignore during mapping
func (m *Mapper[TDto, TModel]) WithIgnoreField(field string) *Mapper[TDto, TModel] {
	m.ignoreFields[field] = true
	return m
}

// WithValidation adds a validation function to be run before DTO to model conversion
func (m *Mapper[TDto, TModel]) WithValidation(fn func(dto TDto) error) *Mapper[TDto, TModel] {
	m.validationFn = fn
	return m
}

// WithStrictMode sets whether the mapper should fail on missing fields
func (m *Mapper[TDto, TModel]) WithStrictMode(strict bool) *Mapper[TDto, TModel] {
	m.strictMode = strict
	return m
}

// WithOptions sets custom options for the mapper
func (m *Mapper[TDto, TModel]) WithOptions(opts *Options) *Mapper[TDto, TModel] {
	if opts != nil {
		m.options = opts
	}
	return m
}

// ToModel converts a DTO to a model
func (m *Mapper[TDto, TModel]) ToModel(dto TDto) (TModel, error) {
	// Run validation if provided
	if m.validationFn != nil {
		if err := m.validationFn(dto); err != nil {
			var zero TModel

			// Convert ValidationErrors to errx if possible
			if valErrors, ok := err.(ValidationErrors); ok {
				return zero, valErrors.ToErrx()
			}

			// Otherwise wrap in an errx error
			return zero, ErrorRegistry.NewWithCause(ErrValidationFailed, err)
		}
	}

	// If custom function provided, use it
	if m.dtoToModelFn != nil {
		return m.dtoToModelFn(dto)
	}

	// Otherwise use reflection-based mapping
	return m.reflectDtoToModel(dto)
}

// ToDto converts a model to a DTO
func (m *Mapper[TDto, TModel]) ToDto(model TModel) (TDto, error) {
	// If custom function provided, use it
	if m.modelToDtoFn != nil {
		return m.modelToDtoFn(model)
	}

	// Otherwise use reflection-based mapping
	return m.reflectModelToDto(model)
}

// reflectDtoToModel implements reflection-based mapping from DTO to model
func (m *Mapper[TDto, TModel]) reflectDtoToModel(dto TDto) (TModel, error) {
	var model TModel
	modelValue := reflect.ValueOf(&model).Elem()
	dtoValue := reflect.ValueOf(dto)

	// Handle nil or zero DTO
	if !dtoValue.IsValid() || dtoValue.IsZero() {
		return model, nil
	}

	// Get DTO type for field mapping
	dtoType := dtoValue.Type()
	if dtoType.Kind() == reflect.Ptr {
		// If DTO is a pointer, dereference it
		if dtoValue.IsNil() {
			return model, nil
		}
		dtoValue = dtoValue.Elem()
		dtoType = dtoValue.Type()
	}

	// Get model type for field mapping
	modelType := modelValue.Type()

	// Map each field from DTO to model
	for i := 0; i < dtoType.NumField(); i++ {
		dtoField := dtoType.Field(i)
		dtoFieldName := dtoField.Name

		// Skip ignored fields
		if m.ignoreFields[dtoFieldName] {
			continue
		}

		// Check if there's a mapping for this field
		modelFieldName, hasMappedName := m.fieldMappings[dtoFieldName]
		if !hasMappedName {
			modelFieldName = dtoFieldName
		}

		// Find the corresponding field in the model
		modelField, fieldFound := findField(modelType, modelFieldName)
		if !fieldFound {
			if m.strictMode {
				return model, ErrorRegistry.New(ErrFieldNotFound).
					WithDetail("field", modelFieldName).
					WithDetail("type", modelType.Name())
			}
			continue
		}

		// Get the value from the DTO
		dtoFieldValue := dtoValue.Field(i)
		if !dtoFieldValue.IsValid() || !dtoFieldValue.CanInterface() {
			continue
		}

		// Get the model field to set
		modelFieldValue := modelValue.FieldByName(modelField.Name)
		if !modelFieldValue.IsValid() || !modelFieldValue.CanSet() {
			if m.strictMode {
				return model, ErrorRegistry.New(ErrCannotSetField).
					WithDetail("field", modelField.Name).
					WithDetail("type", modelType.Name())
			}
			continue
		}

		// Try to set the value
		if err := setFieldValue(modelFieldValue, dtoFieldValue); err != nil {
			if m.strictMode {
				var xerr *errx.Error
				if errors.As(err, &xerr) {
					// If it's already an errx.Error, add more context and pass it through
					return model, xerr.WithDetail("field", modelField.Name)
				}

				// Otherwise wrap in an errx.Error
				return model, ErrorRegistry.NewWithCause(ErrTypeConversion, err).
					WithDetail("field", modelField.Name).
					WithDetail("source_type", dtoFieldValue.Type().String()).
					WithDetail("target_type", modelFieldValue.Type().String())
			}
		}
	}

	return model, nil
}

// reflectModelToDto implements reflection-based mapping from model to DTO
func (m *Mapper[TDto, TModel]) reflectModelToDto(model TModel) (TDto, error) {
	var dto TDto
	dtoValue := reflect.ValueOf(&dto).Elem()
	modelValue := reflect.ValueOf(model)

	// Handle nil or zero model
	if !modelValue.IsValid() || modelValue.IsZero() {
		return dto, nil
	}

	// Get model type for field mapping
	modelType := modelValue.Type()
	if modelType.Kind() == reflect.Ptr {
		// If model is a pointer, dereference it
		if modelValue.IsNil() {
			return dto, nil
		}
		modelValue = modelValue.Elem()
		modelType = modelValue.Type()
	}

	// Get DTO type for field mapping
	dtoType := dtoValue.Type()

	// Reverse field mappings for model-to-dto
	reverseMappings := make(map[string]string)
	for dtoField, modelField := range m.fieldMappings {
		reverseMappings[modelField] = dtoField
	}

	// Map each field from model to DTO
	for i := 0; i < dtoType.NumField(); i++ {
		dtoField := dtoType.Field(i)
		dtoFieldName := dtoField.Name

		// Skip ignored fields
		if m.ignoreFields[dtoFieldName] {
			continue
		}

		// Check if there's a reverse mapping for this field
		modelFieldName, hasMappedName := reverseMappings[dtoFieldName]
		if !hasMappedName {
			// No reverse mapping, use the DTO field name as model field name
			modelFieldName = dtoFieldName
		}

		// Find the corresponding field in the model
		modelField, fieldFound := findField(modelType, modelFieldName)
		if !fieldFound {
			if m.strictMode {
				return dto, fmt.Errorf("field %s not found in model type %s", modelFieldName, modelType.Name())
			}
			continue
		}
		// Get the value from the model
		modelFieldValue := modelValue.FieldByName(modelField.Name)
		if !modelFieldValue.IsValid() || !modelFieldValue.CanInterface() {
			continue
		}

		// Get the DTO field to set
		dtoFieldValue := dtoValue.Field(i)
		if !dtoFieldValue.IsValid() || !dtoFieldValue.CanSet() {
			if m.strictMode {
				return dto, fmt.Errorf("cannot set field %s in DTO type %s", dtoField.Name, dtoType.Name())
			}
			continue
		}

		// Try to set the value
		if err := setFieldValue(dtoFieldValue, modelFieldValue); err != nil {
			if m.strictMode {
				return dto, fmt.Errorf("failed to set field %s: %w", dtoField.Name, err)
			}
		}
	}

	return dto, nil
}

// Helper function to find a field in a struct type
func findField(t reflect.Type, name string) (reflect.StructField, bool) {
	return t.FieldByName(name)
}

// Helper function to set a field value, handling type conversions
func setFieldValue(dst, src reflect.Value) error {
	// Handle nil source
	if !src.IsValid() {
		return nil
	}

	// If types are identical, just set the value
	if dst.Type() == src.Type() {
		dst.Set(src)
		return nil
	}

	// Try to convert the value
	if src.Type().ConvertibleTo(dst.Type()) {
		dst.Set(src.Convert(dst.Type()))
		return nil
	}

	return ErrorRegistry.New(ErrTypeConversion).
		WithDetail("source_type", src.Type().String()).
		WithDetail("destination_type", dst.Type().String())
}
