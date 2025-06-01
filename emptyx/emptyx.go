package emptyx

import (
	"reflect"
)

// Empty checks if a value is considered "empty" based on its type
func Empty(v any) bool {
	if v == nil {
		return true
	}

	val := reflect.ValueOf(v)
	return isEmptyValue(val)
}

// isEmptyValue checks if a reflect.Value is empty
func isEmptyValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Invalid:
		return true
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Complex64, reflect.Complex128:
		return v.Complex() == 0
	case reflect.Array, reflect.Slice:
		return v.Len() == 0
	case reflect.Map:
		return v.Len() == 0 || v.IsNil()
	case reflect.String:
		return v.String() == ""
	case reflect.Ptr, reflect.Interface:
		return v.IsNil()
	case reflect.Chan:
		return v.IsNil()
	case reflect.Func:
		return v.IsNil()
	case reflect.Struct:
		// For structs, check if all fields are empty
		for i := 0; i < v.NumField(); i++ {
			if !isEmptyValue(v.Field(i)) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

// Type-specific functions for better performance when you know the type

// Slice checks if a slice is nil or has length 0
func Slice[T any](slice []T) bool {
	return len(slice) == 0
}

// Map checks if a map is nil or has length 0
func Map[K comparable, V any](m map[K]V) bool {
	return len(m) == 0
}

// String checks if a string is empty
func String(s string) bool {
	return s == ""
}

// Pointer checks if a pointer is nil
func Pointer[T any](ptr *T) bool {
	return ptr == nil
}

// Channel checks if a channel is nil
func Channel[T any](ch chan T) bool {
	return ch == nil
}

// Array checks if all elements in an array are zero values
func Array[T comparable](arr []T) bool {
	var zero T
	for _, v := range arr {
		if v != zero {
			return false
		}
	}
	return true
}

// Nil is an alias for Empty for semantic clarity when checking for nil values
func Nil(v any) bool {
	return Empty(v)
}

// Zero checks if a value is its zero value
func Zero(v any) bool {
	return Empty(v)
}
