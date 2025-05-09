package dtox

import (
	"reflect"
	"strings"
	"unicode"
)

// FieldMappingMode defines strategies for field name matching
type FieldMappingMode int

const (
	// ExactMatch requires field names to match exactly
	ExactMatch FieldMappingMode = iota

	// CaseInsensitiveMatch matches field names ignoring case
	CaseInsensitiveMatch

	// SnakeToCamelMatch converts snake_case to camelCase for matching
	SnakeToCamelMatch

	// CamelToSnakeMatch converts camelCase to snake_case for matching
	CamelToSnakeMatch

	// FlexibleMatch tries multiple strategies for matching
	FlexibleMatch
)

// AutoMapFields automatically creates field mappings between DTO and model types
// based on the specified mapping mode
func (m *Mapper[TDto, TModel]) AutoMapFields(mode FieldMappingMode) *Mapper[TDto, TModel] {
	var dtoType TDto
	var modelType TModel

	dtoFields := getTypeFields(dtoType)
	modelFields := getTypeFields(modelType)

	// Build maps for quick lookup
	modelFieldMap := make(map[string]string)
	for _, field := range modelFields {
		modelFieldMap[field] = field
	}

	// Create mappings based on the specified mode
	for _, dtoField := range dtoFields {
		if m.ignoreFields[dtoField] {
			continue
		}

		var modelField string
		var found bool

		switch mode {
		case ExactMatch:
			modelField, found = modelFieldMap[dtoField]

		case CaseInsensitiveMatch:
			for mField := range modelFieldMap {
				if strings.EqualFold(mField, dtoField) {
					modelField = mField
					found = true
					break
				}
			}

		case SnakeToCamelMatch:
			camelField := snakeToCamel(dtoField)
			if mField, ok := modelFieldMap[camelField]; ok {
				modelField = mField
				found = true
			}

		case CamelToSnakeMatch:
			snakeField := camelToSnake(dtoField)
			if mField, ok := modelFieldMap[snakeField]; ok {
				modelField = mField
				found = true
			}

		case FlexibleMatch:
			// Try exact match first
			if mField, ok := modelFieldMap[dtoField]; ok {
				modelField = mField
				found = true
			} else {
				// Try case-insensitive
				for mField := range modelFieldMap {
					if strings.EqualFold(mField, dtoField) {
						modelField = mField
						found = true
						break
					}
				}

				if !found {
					// Try snake to camel
					camelField := snakeToCamel(dtoField)
					if mField, ok := modelFieldMap[camelField]; ok {
						modelField = mField
						found = true
					} else {
						// Try camel to snake
						snakeField := camelToSnake(dtoField)
						if mField, ok := modelFieldMap[snakeField]; ok {
							modelField = mField
							found = true
						}
					}
				}
			}
		}

		if found && modelField != dtoField {
			m.WithFieldMapping(dtoField, modelField)
		}
	}

	return m
}

// Helper to get all field names from a type
func getTypeFields(obj interface{}) []string {
	t := reflect.TypeOf(obj)

	// Handle pointers
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	// Only process structs
	if t.Kind() != reflect.Struct {
		return []string{}
	}

	fields := make([]string, 0, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		fields = append(fields, field.Name)
	}

	return fields
}

// snakeToCamel converts snake_case to camelCase
func snakeToCamel(s string) string {
	words := strings.Split(s, "_")

	for i := 1; i < len(words); i++ {
		if len(words[i]) > 0 {
			words[i] = string(unicode.ToUpper(rune(words[i][0]))) + words[i][1:]
		}
	}

	return strings.Join(words, "")
}

// camelToSnake converts camelCase to snake_case
func camelToSnake(s string) string {
	var result strings.Builder

	for i, r := range s {
		if unicode.IsUpper(r) {
			if i > 0 {
				result.WriteRune('_')
			}
			result.WriteRune(unicode.ToLower(r))
		} else {
			result.WriteRune(r)
		}
	}

	return result.String()
}
