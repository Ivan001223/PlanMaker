package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// JSONList 用于存储 JSON 数组字段
type JSONList []interface{}

func (j JSONList) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

func (j *JSONList) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}
	var bytes []byte
	switch v := value.(type) {
	case string:
		bytes = []byte(v)
	case []byte:
		bytes = v
	default:
		return fmt.Errorf("unsupported type: %T", value)
	}
	return json.Unmarshal(bytes, j)
}

// JSONMap 用于存储 JSON 对象字段
type JSONMap map[string]interface{}

func (j JSONMap) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

func (j *JSONMap) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}
	var bytes []byte
	switch v := value.(type) {
	case string:
		bytes = []byte(v)
	case []byte:
		bytes = v
	default:
		return fmt.Errorf("unsupported type: %T", value)
	}
	return json.Unmarshal(bytes, j)
}
