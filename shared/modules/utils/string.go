package utils

import (
	"math/rand"
	"reflect"
	"strings"
)

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ123456789")

func GenerateRandomStringWithLength(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

// TrimAllStringFields iterates over all fields of the input and trims any string fields.
// Input: any (interface{}) - can be a struct, pointer to struct, slice, map, etc.
// Output: same type as input but with all string fields trimmed.
func TrimAllStringFields(input any) any {
	if input == nil {
		return nil
	}

	value := reflect.ValueOf(input)
	return trimValue(value).Interface()
}

// trimValue recursively trims string fields for various data types.
func trimValue(v reflect.Value) reflect.Value {
	// Handle pointer
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return v
		}
		// Create a new pointer with the trimmed value
		elem := v.Elem()
		newElem := trimValue(elem)
		
		// Create a new pointer pointing to the trimmed value
		newPtr := reflect.New(newElem.Type())
		newPtr.Elem().Set(newElem)
		return newPtr
	}

	// Handle struct
	if v.Kind() == reflect.Struct {
		newStruct := reflect.New(v.Type()).Elem()
		
		for i := 0; i < v.NumField(); i++ {
			field := v.Field(i)
			fieldType := v.Type().Field(i)
			
			// Skip unexported (private) fields
			if !fieldType.IsExported() {
				continue
			}
			
			// If the field is settable
			if newStruct.Field(i).CanSet() {
				trimmedField := trimValue(field)
				newStruct.Field(i).Set(trimmedField)
			}
		}
		return newStruct
	}

	// Handle slice
	if v.Kind() == reflect.Slice {
		if v.IsNil() {
			return v
		}
		newSlice := reflect.MakeSlice(v.Type(), v.Len(), v.Cap())
		for i := 0; i < v.Len(); i++ {
			trimmedElem := trimValue(v.Index(i))
			newSlice.Index(i).Set(trimmedElem)
		}
		return newSlice
	}

	// Handle map
	if v.Kind() == reflect.Map {
		if v.IsNil() {
			return v
		}
		newMap := reflect.MakeMap(v.Type())
		iter := v.MapRange()
		for iter.Next() {
			key := iter.Key()
			val := iter.Value()
			
			// Trim both key and value if they are strings
			trimmedKey := trimValue(key)
			trimmedVal := trimValue(val)
			newMap.SetMapIndex(trimmedKey, trimmedVal)
		}
		return newMap
	}

	// Handle string – the core case
	if v.Kind() == reflect.String {
		trimmed := strings.TrimSpace(v.String())
		return reflect.ValueOf(trimmed)
	}

	// For other types (int, bool, float, etc.), return as is
	return v
}