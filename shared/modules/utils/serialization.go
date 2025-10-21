package utils

import (
	"encoding/json"
	"fmt"
	"reflect"
)

// SerializeModel converts any model to []byte using JSON serialization
// This function provides a generic way to serialize any struct to JSON bytes
// for storage in Redis or other byte-based storage systems.
//
// Parameters:
//   - model: Any type T that can be JSON marshaled
//
// Returns:
//   - []byte: JSON representation of the model
//   - error: Error if serialization fails or if model is nil pointer
//
// Example usage:
//
//	policy := &models.BasePolicy{...}
//	data, err := SerializeModel(policy)
//	if err != nil {
//	    return fmt.Errorf("failed to serialize policy: %w", err)
//	}
func SerializeModel[T any](model T) ([]byte, error) {
	// Check if the model is a nil pointer
	value := reflect.ValueOf(model)
	if value.Kind() == reflect.Pointer && value.IsNil() {
		return nil, fmt.Errorf("cannot serialize nil pointer")
	}

	// Serialize the model to JSON
	data, err := json.Marshal(model)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal model: %w", err)
	}

	return data, nil
}

// DeserializeModel converts []byte back to a model of type T
// This is the inverse operation of SerializeModel for retrieving data
// from Redis or other byte-based storage systems.
//
// Parameters:
//   - data: JSON byte data to deserialize
//   - target: Pointer to the target type to deserialize into
//
// Returns:
//   - error: Error if deserialization fails
//
// Example usage:
//
//	var policy models.BasePolicy
//	err := DeserializeModel(data, &policy)
//	if err != nil {
//	    return fmt.Errorf("failed to deserialize policy: %w", err)
//	}
func DeserializeModel[T any](data []byte, target *T) error {
	if len(data) == 0 {
		return fmt.Errorf("cannot deserialize empty data")
	}

	if target == nil {
		return fmt.Errorf("target cannot be nil")
	}

	err := json.Unmarshal(data, target)
	if err != nil {
		return fmt.Errorf("failed to unmarshal data: %w", err)
	}

	return nil
}
