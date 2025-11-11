package utils

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

type JSONMap map[string]any

func (j JSONMap) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil // Store NULL if the map is nil
	}
	return json.Marshal(j)
}

func (j *JSONMap) Scan(value any) error {
	if value == nil {
		*j = nil
		return nil
	}

	b, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("JSONMap: Scan failed, expected []byte but got %T", value)
	}

	return json.Unmarshal(b, j)
}

func (j *JSONMap) KeySlice() []string {
	res := []string{}
	for k := range *j {
		res = append(res, k)
	}
	return res
}
