package utils

import (
	"fmt"
	"reflect"
	"sync"
	"time"
)

// FieldMapping caches reflection info for performance
type FieldMapping struct {
	DestFieldIndex int
	SrcFieldName   string
	IsNested       bool
	NestedFields   []NestedFieldMapping
}

type NestedFieldMapping struct {
	DestFieldIndex int
	SrcFieldName   string
}

var (
	mappingCache = make(map[string][]FieldMapping)
	cacheMutex   sync.RWMutex
)

// FastAssembleWithPrefix uses cached reflection info for better performance
func FastAssembleWithPrefix[T any](dest *T, src any, prefix string) error {
	destType := reflect.TypeOf(dest).Elem()
	srcType := reflect.TypeOf(src)
	if srcType.Kind() == reflect.Pointer {
		srcType = srcType.Elem()
	}

	cacheKey := fmt.Sprintf("%s_%s_%s", destType.String(), srcType.String(), prefix)

	// Check cache first
	cacheMutex.RLock()
	mappings, exists := mappingCache[cacheKey]
	cacheMutex.RUnlock()

	if !exists {
		// Build mappings and cache them
		mappings = buildFieldMappings(destType, srcType, prefix)

		cacheMutex.Lock()
		mappingCache[cacheKey] = mappings
		cacheMutex.Unlock()
	}

	// Apply cached mappings
	return applyMappings(dest, src, mappings)
}

func buildFieldMappings(destType, srcType reflect.Type, prefix string) []FieldMapping {
	var mappings []FieldMapping

	// Build source field map
	srcFieldMap := make(map[string]int)
	for i := range srcType.NumField() {
		field := srcType.Field(i)
		dbTag := field.Tag.Get("db")
		if dbTag != "" && dbTag != "-" {
			srcFieldMap[dbTag] = i
		}
	}

	// Build destination mappings
	for i := range destType.NumField() {
		destField := destType.Field(i)

		if destField.Type.Kind() == reflect.Struct &&
			!isTimeType(destField.Type) &&
			!isUUIDType(destField.Type) {
			// Handle nested struct
			nestedMappings := buildNestedMappings(destField.Type, srcFieldMap, prefix)
			mappings = append(mappings, FieldMapping{
				DestFieldIndex: i,
				IsNested:       true,
				NestedFields:   nestedMappings,
			})
		} else {
			// Handle regular field
			dbTag := destField.Tag.Get("db")
			if dbTag != "" && dbTag != "-" {
				if srcIndex, exists := srcFieldMap[dbTag]; exists {
					mappings = append(mappings, FieldMapping{
						DestFieldIndex: i,
						SrcFieldName:   dbTag,
						IsNested:       false,
					})
					_ = srcIndex // Use the index when applying mappings
				}
			}
		}
	}

	return mappings
}

func buildNestedMappings(nestedType reflect.Type, srcFieldMap map[string]int, prefix string) []NestedFieldMapping {
	var nestedMappings []NestedFieldMapping

	for i := range nestedType.NumField() {
		field := nestedType.Field(i)
		dbTag := field.Tag.Get("db")
		if dbTag != "" && dbTag != "-" {
			prefixedName := prefix + dbTag
			if _, exists := srcFieldMap[prefixedName]; exists {
				nestedMappings = append(nestedMappings, NestedFieldMapping{
					DestFieldIndex: i,
					SrcFieldName:   prefixedName,
				})
			}
		}
	}

	return nestedMappings
}

func applyMappings[T any](dest *T, src any, mappings []FieldMapping) error {
	destValue := reflect.ValueOf(dest).Elem()
	srcValue := reflect.ValueOf(src)
	if srcValue.Kind() == reflect.Pointer {
		srcValue = srcValue.Elem()
	}

	// Create source field map for quick lookup
	srcFields := make(map[string]reflect.Value)
	srcType := srcValue.Type()
	for i := range srcValue.NumField() {
		field := srcType.Field(i)
		dbTag := field.Tag.Get("db")
		if dbTag != "" && dbTag != "-" {
			srcFields[dbTag] = srcValue.Field(i)
		}
	}

	for _, mapping := range mappings {
		destField := destValue.Field(mapping.DestFieldIndex)

		if mapping.IsNested {
			// Handle nested struct
			nestedStruct := reflect.New(destField.Type()).Elem()

			for _, nestedMapping := range mapping.NestedFields {
				nestedField := nestedStruct.Field(nestedMapping.DestFieldIndex)
				if srcField, exists := srcFields[nestedMapping.SrcFieldName]; exists {
					if err := assignField(nestedField, srcField); err != nil {
						return err
					}
				}
			}

			destField.Set(nestedStruct)
		} else {
			// Handle regular field
			if srcField, exists := srcFields[mapping.SrcFieldName]; exists {
				if err := assignField(destField, srcField); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func assignField(dest, src reflect.Value) error {
	if !src.IsValid() {
		return nil // Skip invalid source values
	}

	srcType := src.Type()
	destType := dest.Type()

	// Handle direct assignment for same types
	if srcType.AssignableTo(destType) {
		dest.Set(src)
		return nil
	}

	// Handle convertible types
	if srcType.ConvertibleTo(destType) {
		dest.Set(src.Convert(destType))
		return nil
	}

	// Handle pointer types
	if destType.Kind() == reflect.Pointer {
		if src.IsZero() {
			return nil // Leave as nil
		}

		elemType := destType.Elem()
		if srcType.AssignableTo(elemType) {
			newVal := reflect.New(elemType)
			newVal.Elem().Set(src)
			dest.Set(newVal)
			return nil
		}
	}

	return fmt.Errorf("cannot assign %s to %s", srcType, destType)
}

// Helper functions for type checking
func isTimeType(t reflect.Type) bool {
	return t == reflect.TypeOf(time.Time{})
}

func isUUIDType(t reflect.Type) bool {
	// Assuming UUID is imported as github.com/google/uuid
	return t.String() == "uuid.UUID"
}
