package utils

import (
	"encoding/json"
)

func MapToStruct(mapData map[string]any, result any) error {
	jsonData, err := json.Marshal(mapData)
	if err != nil {
		
		return err
	}
	err = json.Unmarshal(jsonData, result)
	if err != nil {
		
		return err
	}
	return nil
}
