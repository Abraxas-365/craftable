package clone

import (
	"reflect"
)

// Struct creates a deep copy of any struct
func Struct(v interface{}) interface{} {
	// Check if the input is a struct
	if reflect.TypeOf(v).Kind() != reflect.Struct {
		return v
	}

	// Create a new instance of the same type as the input
	clone := reflect.New(reflect.TypeOf(v)).Elem()

	// Get the value of the input
	value := reflect.ValueOf(v)

	// Copy all fields
	for i := 0; i < value.NumField(); i++ {
		field := value.Field(i)
		cloneField := clone.Field(i)

		// Handle pointer fields
		if field.Kind() == reflect.Ptr && !field.IsNil() {
			cloneField.Set(reflect.New(field.Elem().Type()))
			cloneField.Elem().Set(reflect.ValueOf(Struct(field.Elem().Interface())))
		} else if field.Kind() == reflect.Struct {
			// Recursively clone nested structs
			cloneField.Set(reflect.ValueOf(Struct(field.Interface())))
		} else {
			cloneField.Set(field)
		}
	}

	return clone.Interface()
}

// Map creates a deep copy of a map
func Map(m interface{}) interface{} {
	// Check if the input is a map
	if reflect.TypeOf(m).Kind() != reflect.Map {
		return m
	}

	originalValue := reflect.ValueOf(m)
	cloneValue := reflect.MakeMap(originalValue.Type())

	for _, key := range originalValue.MapKeys() {
		originalElem := originalValue.MapIndex(key)
		cloneElem := reflect.New(originalElem.Type()).Elem()

		// Handle nested maps and structs
		switch originalElem.Kind() {
		case reflect.Map:
			cloneElem.Set(reflect.ValueOf(Map(originalElem.Interface())))
		case reflect.Struct:
			cloneElem.Set(reflect.ValueOf(Struct(originalElem.Interface())))
		default:
			cloneElem.Set(originalElem)
		}

		cloneValue.SetMapIndex(key, cloneElem)
	}

	return cloneValue.Interface()
}
