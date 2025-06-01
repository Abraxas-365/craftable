package ptrx

import (
	"time"
)

// Bool returns a pointer value for the bool value passed in.
func Bool(v bool) *bool {
	return &v
}

// BoolSlice returns a slice of bool pointers from the values
// passed in.
func BoolSlice(vs []bool) []*bool {
	ps := make([]*bool, len(vs))
	for i, v := range vs {
		vv := v
		ps[i] = &vv
	}

	return ps
}

// BoolMap returns a map of bool pointers from the values
// passed in.
func BoolMap(vs map[string]bool) map[string]*bool {
	ps := make(map[string]*bool, len(vs))
	for k, v := range vs {
		vv := v
		ps[k] = &vv
	}

	return ps
}

// Byte returns a pointer value for the byte value passed in.
func Byte(v byte) *byte {
	return &v
}

// ByteSlice returns a slice of byte pointers from the values
// passed in.
func ByteSlice(vs []byte) []*byte {
	ps := make([]*byte, len(vs))
	for i, v := range vs {
		vv := v
		ps[i] = &vv
	}

	return ps
}

// ByteMap returns a map of byte pointers from the values
// passed in.
func ByteMap(vs map[string]byte) map[string]*byte {
	ps := make(map[string]*byte, len(vs))
	for k, v := range vs {
		vv := v
		ps[k] = &vv
	}

	return ps
}

// String returns a pointer value for the string value passed in.
func String(v string) *string {
	return &v
}

// StringSlice returns a slice of string pointers from the values
// passed in.
func StringSlice(vs []string) []*string {
	ps := make([]*string, len(vs))
	for i, v := range vs {
		vv := v
		ps[i] = &vv
	}

	return ps
}

// StringMap returns a map of string pointers from the values
// passed in.
func StringMap(vs map[string]string) map[string]*string {
	ps := make(map[string]*string, len(vs))
	for k, v := range vs {
		vv := v
		ps[k] = &vv
	}

	return ps
}

// Int returns a pointer value for the int value passed in.
func Int(v int) *int {
	return &v
}

// IntSlice returns a slice of int pointers from the values
// passed in.
func IntSlice(vs []int) []*int {
	ps := make([]*int, len(vs))
	for i, v := range vs {
		vv := v
		ps[i] = &vv
	}

	return ps
}

// IntMap returns a map of int pointers from the values
// passed in.
func IntMap(vs map[string]int) map[string]*int {
	ps := make(map[string]*int, len(vs))
	for k, v := range vs {
		vv := v
		ps[k] = &vv
	}

	return ps
}

// Int8 returns a pointer value for the int8 value passed in.
func Int8(v int8) *int8 {
	return &v
}

// Int8Slice returns a slice of int8 pointers from the values
// passed in.
func Int8Slice(vs []int8) []*int8 {
	ps := make([]*int8, len(vs))
	for i, v := range vs {
		vv := v
		ps[i] = &vv
	}

	return ps
}

// Int8Map returns a map of int8 pointers from the values
// passed in.
func Int8Map(vs map[string]int8) map[string]*int8 {
	ps := make(map[string]*int8, len(vs))
	for k, v := range vs {
		vv := v
		ps[k] = &vv
	}

	return ps
}

// Int16 returns a pointer value for the int16 value passed in.
func Int16(v int16) *int16 {
	return &v
}

// Int16Slice returns a slice of int16 pointers from the values
// passed in.
func Int16Slice(vs []int16) []*int16 {
	ps := make([]*int16, len(vs))
	for i, v := range vs {
		vv := v
		ps[i] = &vv
	}

	return ps
}

// Int16Map returns a map of int16 pointers from the values
// passed in.
func Int16Map(vs map[string]int16) map[string]*int16 {
	ps := make(map[string]*int16, len(vs))
	for k, v := range vs {
		vv := v
		ps[k] = &vv
	}

	return ps
}

// Int32 returns a pointer value for the int32 value passed in.
func Int32(v int32) *int32 {
	return &v
}

// Int32Slice returns a slice of int32 pointers from the values
// passed in.
func Int32Slice(vs []int32) []*int32 {
	ps := make([]*int32, len(vs))
	for i, v := range vs {
		vv := v
		ps[i] = &vv
	}

	return ps
}

// Int32Map returns a map of int32 pointers from the values
// passed in.
func Int32Map(vs map[string]int32) map[string]*int32 {
	ps := make(map[string]*int32, len(vs))
	for k, v := range vs {
		vv := v
		ps[k] = &vv
	}

	return ps
}

// Int64 returns a pointer value for the int64 value passed in.
func Int64(v int64) *int64 {
	return &v
}

// Int64Slice returns a slice of int64 pointers from the values
// passed in.
func Int64Slice(vs []int64) []*int64 {
	ps := make([]*int64, len(vs))
	for i, v := range vs {
		vv := v
		ps[i] = &vv
	}

	return ps
}

// Int64Map returns a map of int64 pointers from the values
// passed in.
func Int64Map(vs map[string]int64) map[string]*int64 {
	ps := make(map[string]*int64, len(vs))
	for k, v := range vs {
		vv := v
		ps[k] = &vv
	}

	return ps
}

// Uint returns a pointer value for the uint value passed in.
func Uint(v uint) *uint {
	return &v
}

// UintSlice returns a slice of uint pointers from the values
// passed in.
func UintSlice(vs []uint) []*uint {
	ps := make([]*uint, len(vs))
	for i, v := range vs {
		vv := v
		ps[i] = &vv
	}

	return ps
}

// UintMap returns a map of uint pointers from the values
// passed in.
func UintMap(vs map[string]uint) map[string]*uint {
	ps := make(map[string]*uint, len(vs))
	for k, v := range vs {
		vv := v
		ps[k] = &vv
	}

	return ps
}

// Uint8 returns a pointer value for the uint8 value passed in.
func Uint8(v uint8) *uint8 {
	return &v
}

// Uint8Slice returns a slice of uint8 pointers from the values
// passed in.
func Uint8Slice(vs []uint8) []*uint8 {
	ps := make([]*uint8, len(vs))
	for i, v := range vs {
		vv := v
		ps[i] = &vv
	}

	return ps
}

// Uint8Map returns a map of uint8 pointers from the values
// passed in.
func Uint8Map(vs map[string]uint8) map[string]*uint8 {
	ps := make(map[string]*uint8, len(vs))
	for k, v := range vs {
		vv := v
		ps[k] = &vv
	}

	return ps
}

// Uint16 returns a pointer value for the uint16 value passed in.
func Uint16(v uint16) *uint16 {
	return &v
}

// Uint16Slice returns a slice of uint16 pointers from the values
// passed in.
func Uint16Slice(vs []uint16) []*uint16 {
	ps := make([]*uint16, len(vs))
	for i, v := range vs {
		vv := v
		ps[i] = &vv
	}

	return ps
}

// Uint16Map returns a map of uint16 pointers from the values
// passed in.
func Uint16Map(vs map[string]uint16) map[string]*uint16 {
	ps := make(map[string]*uint16, len(vs))
	for k, v := range vs {
		vv := v
		ps[k] = &vv
	}

	return ps
}

// Uint32 returns a pointer value for the uint32 value passed in.
func Uint32(v uint32) *uint32 {
	return &v
}

// Uint32Slice returns a slice of uint32 pointers from the values
// passed in.
func Uint32Slice(vs []uint32) []*uint32 {
	ps := make([]*uint32, len(vs))
	for i, v := range vs {
		vv := v
		ps[i] = &vv
	}

	return ps
}

// Uint32Map returns a map of uint32 pointers from the values
// passed in.
func Uint32Map(vs map[string]uint32) map[string]*uint32 {
	ps := make(map[string]*uint32, len(vs))
	for k, v := range vs {
		vv := v
		ps[k] = &vv
	}

	return ps
}

// Uint64 returns a pointer value for the uint64 value passed in.
func Uint64(v uint64) *uint64 {
	return &v
}

// Uint64Slice returns a slice of uint64 pointers from the values
// passed in.
func Uint64Slice(vs []uint64) []*uint64 {
	ps := make([]*uint64, len(vs))
	for i, v := range vs {
		vv := v
		ps[i] = &vv
	}

	return ps
}

// Uint64Map returns a map of uint64 pointers from the values
// passed in.
func Uint64Map(vs map[string]uint64) map[string]*uint64 {
	ps := make(map[string]*uint64, len(vs))
	for k, v := range vs {
		vv := v
		ps[k] = &vv
	}

	return ps
}

// Float32 returns a pointer value for the float32 value passed in.
func Float32(v float32) *float32 {
	return &v
}

// Float32Slice returns a slice of float32 pointers from the values
// passed in.
func Float32Slice(vs []float32) []*float32 {
	ps := make([]*float32, len(vs))
	for i, v := range vs {
		vv := v
		ps[i] = &vv
	}

	return ps
}

// Float32Map returns a map of float32 pointers from the values
// passed in.
func Float32Map(vs map[string]float32) map[string]*float32 {
	ps := make(map[string]*float32, len(vs))
	for k, v := range vs {
		vv := v
		ps[k] = &vv
	}

	return ps
}

// Float64 returns a pointer value for the float64 value passed in.
func Float64(v float64) *float64 {
	return &v
}

// Float64Slice returns a slice of float64 pointers from the values
// passed in.
func Float64Slice(vs []float64) []*float64 {
	ps := make([]*float64, len(vs))
	for i, v := range vs {
		vv := v
		ps[i] = &vv
	}

	return ps
}

// Float64Map returns a map of float64 pointers from the values
// passed in.
func Float64Map(vs map[string]float64) map[string]*float64 {
	ps := make(map[string]*float64, len(vs))
	for k, v := range vs {
		vv := v
		ps[k] = &vv
	}

	return ps
}

// Time returns a pointer value for the time.Time value passed in.
func Time(v time.Time) *time.Time {
	return &v
}

// TimeSlice returns a slice of time.Time pointers from the values
// passed in.
func TimeSlice(vs []time.Time) []*time.Time {
	ps := make([]*time.Time, len(vs))
	for i, v := range vs {
		vv := v
		ps[i] = &vv
	}

	return ps
}

// TimeMap returns a map of time.Time pointers from the values
// passed in.
func TimeMap(vs map[string]time.Time) map[string]*time.Time {
	ps := make(map[string]*time.Time, len(vs))
	for k, v := range vs {
		vv := v
		ps[k] = &vv
	}

	return ps
}

// Duration returns a pointer value for the time.Duration value passed in.
func Duration(v time.Duration) *time.Duration {
	return &v
}

// DurationSlice returns a slice of time.Duration pointers from the values
// passed in.
func DurationSlice(vs []time.Duration) []*time.Duration {
	ps := make([]*time.Duration, len(vs))
	for i, v := range vs {
		vv := v
		ps[i] = &vv
	}

	return ps
}

// DurationMap returns a map of time.Duration pointers from the values
// passed in.
func DurationMap(vs map[string]time.Duration) map[string]*time.Duration {
	ps := make(map[string]*time.Duration, len(vs))
	for k, v := range vs {
		vv := v
		ps[k] = &vv
	}

	return ps
}

func BoolValue(v *bool) bool {
	if v != nil {
		return *v
	}
	return false
}

// BoolValueOr returns the value of the bool pointer passed in or the default value if the pointer is nil.
func BoolValueOr(v *bool, def bool) bool {
	if v != nil {
		return *v
	}
	return def
}

// ByteValue returns the value of the byte pointer passed in or 0 if the pointer is nil.
func ByteValue(v *byte) byte {
	if v != nil {
		return *v
	}
	return 0
}

// ByteValueOr returns the value of the byte pointer passed in or the default value if the pointer is nil.
func ByteValueOr(v *byte, def byte) byte {
	if v != nil {
		return *v
	}
	return def
}

// StringValue returns the value of the string pointer passed in or empty string if the pointer is nil.
func StringValue(v *string) string {
	if v != nil {
		return *v
	}
	return ""
}

// StringValueOr returns the value of the string pointer passed in or the default value if the pointer is nil.
func StringValueOr(v *string, def string) string {
	if v != nil {
		return *v
	}
	return def
}

// IntValue returns the value of the int pointer passed in or 0 if the pointer is nil.
func IntValue(v *int) int {
	if v != nil {
		return *v
	}
	return 0
}

// IntValueOr returns the value of the int pointer passed in or the default value if the pointer is nil.
func IntValueOr(v *int, def int) int {
	if v != nil {
		return *v
	}
	return def
}

// Int8Value returns the value of the int8 pointer passed in or 0 if the pointer is nil.
func Int8Value(v *int8) int8 {
	if v != nil {
		return *v
	}
	return 0
}

// Int8ValueOr returns the value of the int8 pointer passed in or the default value if the pointer is nil.
func Int8ValueOr(v *int8, def int8) int8 {
	if v != nil {
		return *v
	}
	return def
}

// Int16Value returns the value of the int16 pointer passed in or 0 if the pointer is nil.
func Int16Value(v *int16) int16 {
	if v != nil {
		return *v
	}
	return 0
}

// Int16ValueOr returns the value of the int16 pointer passed in or the default value if the pointer is nil.
func Int16ValueOr(v *int16, def int16) int16 {
	if v != nil {
		return *v
	}
	return def
}

// Int32Value returns the value of the int32 pointer passed in or 0 if the pointer is nil.
func Int32Value(v *int32) int32 {
	if v != nil {
		return *v
	}
	return 0
}

// Int32ValueOr returns the value of the int32 pointer passed in or the default value if the pointer is nil.
func Int32ValueOr(v *int32, def int32) int32 {
	if v != nil {
		return *v
	}
	return def
}

// Int64Value returns the value of the int64 pointer passed in or 0 if the pointer is nil.
func Int64Value(v *int64) int64 {
	if v != nil {
		return *v
	}
	return 0
}

// Int64ValueOr returns the value of the int64 pointer passed in or the default value if the pointer is nil.
func Int64ValueOr(v *int64, def int64) int64 {
	if v != nil {
		return *v
	}
	return def
}

// UintValue returns the value of the uint pointer passed in or 0 if the pointer is nil.
func UintValue(v *uint) uint {
	if v != nil {
		return *v
	}
	return 0
}

// UintValueOr returns the value of the uint pointer passed in or the default value if the pointer is nil.
func UintValueOr(v *uint, def uint) uint {
	if v != nil {
		return *v
	}
	return def
}

// Uint8Value returns the value of the uint8 pointer passed in or 0 if the pointer is nil.
func Uint8Value(v *uint8) uint8 {
	if v != nil {
		return *v
	}
	return 0
}

// Uint8ValueOr returns the value of the uint8 pointer passed in or the default value if the pointer is nil.
func Uint8ValueOr(v *uint8, def uint8) uint8 {
	if v != nil {
		return *v
	}
	return def
}

// Uint16Value returns the value of the uint16 pointer passed in or 0 if the pointer is nil.
func Uint16Value(v *uint16) uint16 {
	if v != nil {
		return *v
	}
	return 0
}

// Uint16ValueOr returns the value of the uint16 pointer passed in or the default value if the pointer is nil.
func Uint16ValueOr(v *uint16, def uint16) uint16 {
	if v != nil {
		return *v
	}
	return def
}

// Uint32Value returns the value of the uint32 pointer passed in or 0 if the pointer is nil.
func Uint32Value(v *uint32) uint32 {
	if v != nil {
		return *v
	}
	return 0
}

// Uint32ValueOr returns the value of the uint32 pointer passed in or the default value if the pointer is nil.
func Uint32ValueOr(v *uint32, def uint32) uint32 {
	if v != nil {
		return *v
	}
	return def
}

// Uint64Value returns the value of the uint64 pointer passed in or 0 if the pointer is nil.
func Uint64Value(v *uint64) uint64 {
	if v != nil {
		return *v
	}
	return 0
}

// Uint64ValueOr returns the value of the uint64 pointer passed in or the default value if the pointer is nil.
func Uint64ValueOr(v *uint64, def uint64) uint64 {
	if v != nil {
		return *v
	}
	return def
}

// Float32Value returns the value of the float32 pointer passed in or 0 if the pointer is nil.
func Float32Value(v *float32) float32 {
	if v != nil {
		return *v
	}
	return 0
}

// Float32ValueOr returns the value of the float32 pointer passed in or the default value if the pointer is nil.
func Float32ValueOr(v *float32, def float32) float32 {
	if v != nil {
		return *v
	}
	return def
}

// Float64Value returns the value of the float64 pointer passed in or 0 if the pointer is nil.
func Float64Value(v *float64) float64 {
	if v != nil {
		return *v
	}
	return 0
}

// Float64ValueOr returns the value of the float64 pointer passed in or the default value if the pointer is nil.
func Float64ValueOr(v *float64, def float64) float64 {
	if v != nil {
		return *v
	}
	return def
}

// TimeValue returns the value of the time.Time pointer passed in or zero time if the pointer is nil.
func TimeValue(v *time.Time) time.Time {
	if v != nil {
		return *v
	}
	return time.Time{}
}

// TimeValueOr returns the value of the time.Time pointer passed in or the default value if the pointer is nil.
func TimeValueOr(v *time.Time, def time.Time) time.Time {
	if v != nil {
		return *v
	}
	return def
}

// DurationValue returns the value of the time.Duration pointer passed in or 0 if the pointer is nil.
func DurationValue(v *time.Duration) time.Duration {
	if v != nil {
		return *v
	}
	return 0
}

// DurationValueOr returns the value of the time.Duration pointer passed in or the default value if the pointer is nil.
func DurationValueOr(v *time.Duration, def time.Duration) time.Duration {
	if v != nil {
		return *v
	}
	return def
}

// Generic functions for any type (Go 1.18+)

// Value returns the value of the pointer passed in or the zero value if the pointer is nil.
func Value[T any](v *T) T {
	if v != nil {
		return *v
	}
	var zero T
	return zero
}

// ValueOr returns the value of the pointer passed in or the default value if the pointer is nil.
func ValueOr[T any](v *T, def T) T {
	if v != nil {
		return *v
	}
	return def
}

// IsNil checks if a pointer is nil.
func IsNil[T any](v *T) bool {
	return v == nil
}

// IsNotNil checks if a pointer is not nil.
func IsNotNil[T any](v *T) bool {
	return v != nil
}
