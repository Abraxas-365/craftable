package dtox

import (
	"fmt"
	"reflect"
)

// WithPartial configures the mapper to handle partial updates
func (m *Mapper[TDto, TModel]) WithPartial(isPartialUpdate bool) *Mapper[TDto, TModel] {
	if isPartialUpdate {
		// Override the DTO to model function to handle partial updates
		originalFn := m.dtoToModelFn

		m.dtoToModelFn = func(dto TDto) (TModel, error) {
			// If a custom function is already set, use it
			if originalFn != nil {
				return originalFn(dto)
			}

			// Otherwise implement default partial update behavior
			return m.partialDtoToModel(dto)
		}
	}

	return m
}

// partialDtoToModel implements partial updates from DTO to model
func (m *Mapper[TDto, TModel]) partialDtoToModel(dto TDto) (TModel, error) {
	// Use reflection to check which fields are set in the DTO
	dtoValue := reflect.ValueOf(dto)
	if dtoValue.Kind() == reflect.Ptr {
		dtoValue = dtoValue.Elem()
	}

	// Create a zero value of the DTO type for comparison
	zeroDto := reflect.New(dtoValue.Type()).Elem()

	// Create a map of field names to ignore (fields with zero values)
	ignoreFields := make(map[string]bool)

	// Check each field in the DTO
	for i := 0; i < dtoValue.NumField(); i++ {
		field := dtoValue.Type().Field(i)
		fieldValue := dtoValue.Field(i)
		zeroValue := zeroDto.Field(i)

		// If the field has its zero value, add it to ignoreFields
		if reflect.DeepEqual(fieldValue.Interface(), zeroValue.Interface()) {
			ignoreFields[field.Name] = true
		}
	}

	// Create a temporary mapper with the ignore fields
	tempMapper := &Mapper[TDto, TModel]{
		dtoToModelFn:  nil,
		modelToDtoFn:  m.modelToDtoFn,
		fieldMappings: m.fieldMappings,
		ignoreFields:  ignoreFields,
		validationFn:  m.validationFn,
		strictMode:    m.strictMode,
		options:       m.options,
		fieldCache:    m.fieldCache,
	}

	// Use the reflectDtoToModel method with ignore fields set
	return tempMapper.reflectDtoToModel(dto)
}

// ApplyPartialUpdate applies a partial update to an existing model
func (m *Mapper[TDto, TModel]) ApplyPartialUpdate(existingModel TModel, dto TDto) (TModel, error) {
	// Run validation if provided
	if m.validationFn != nil {
		if err := m.validationFn(dto); err != nil {
			return existingModel, fmt.Errorf("validation failed: %w", err)
		}
	}

	// Use reflection to check which fields are set in the DTO
	dtoValue := reflect.ValueOf(dto)
	if dtoValue.Kind() == reflect.Ptr {
		dtoValue = dtoValue.Elem()
	}

	// Create a zero value of the DTO type for comparison
	zeroDto := reflect.New(dtoValue.Type()).Elem()

	// Get the existing model as a reflect.Value
	modelValue := reflect.ValueOf(&existingModel).Elem()

	// Get DTO type for field mapping
	dtoType := dtoValue.Type()

	// Get model type for field mapping
	modelType := modelValue.Type()

	// Update each non-zero field from DTO to model
	for i := 0; i < dtoType.NumField(); i++ {
		dtoField := dtoType.Field(i)
		dtoFieldName := dtoField.Name

		// Skip ignored fields
		if m.ignoreFields[dtoFieldName] {
			continue
		}

		// Get the field value from the DTO
		dtoFieldValue := dtoValue.Field(i)
		zeroFieldValue := zeroDto.Field(i)

		// Skip zero values (not set in partial update)
		if reflect.DeepEqual(dtoFieldValue.Interface(), zeroFieldValue.Interface()) {
			continue
		}

		// Check if there's a mapping for this field
		modelFieldName, hasMappedName := m.fieldMappings[dtoFieldName]
		if !hasMappedName {
			modelFieldName = dtoFieldName
		}

		// Find the corresponding field in the model
		modelField, _ := findField(modelType, modelFieldName)

		// Check if we found the field
		if modelField.Name == "" {
			if m.strictMode {
				return existingModel, fmt.Errorf("field %s not found in model type %s", modelFieldName, modelType.Name())
			}
			continue
		}

		// Skip if field value is not accessible
		if !dtoFieldValue.IsValid() || !dtoFieldValue.CanInterface() {
			continue
		}

		// Get the model field to set
		modelFieldValue := modelValue.FieldByName(modelField.Name)
		if !modelFieldValue.IsValid() || !modelFieldValue.CanSet() {
			if m.strictMode {
				return existingModel, fmt.Errorf("cannot set field %s in model type %s", modelField.Name, modelType.Name())
			}
			continue
		}

		// Update the field
		if err := setFieldValue(modelFieldValue, dtoFieldValue); err != nil {
			if m.strictMode {
				return existingModel, fmt.Errorf("failed to set field %s: %w", modelField.Name, err)
			}
		}
	}

	return existingModel, nil
}
